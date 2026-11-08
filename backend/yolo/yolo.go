// Package yolo is a delegated backend: it hands the mapping from a model to
// a language to a language model, and keeps for itself the prompt, the
// file paths, and the loss the model reports. See CONTEXT.md and #1505.
//
// It is served only as tdl-gen-yolo, so the model client never links into
// tdl itself.
package yolo

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name names this backend in a target block and as tdl-gen-yolo.
const Name = "yolo"

// maxTurns bounds a conversation, so a model that never stops calling
// tools cannot run up a bill.
const maxTurns = 50

// Backend implements [plugin.Backend].
type Backend struct {
	// Provider answers the conversation. Nil means [FromEnv].
	Provider Provider
}

func (Backend) Describe() plugin.Description {
	str := []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_STRING}
	block := plugin.On(plugin.OnBlock)
	return plugin.Description{
		Name:    Name,
		Version: "0.1.0",
		// A language model answers the same request differently, so tdl
		// keeps the files it wrote while the request is unchanged.
		Nondeterministic: true,
		Directives: []*plugin.DirectiveSpec{
			// The language to write, required.
			{Name: "language", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: block},
			// Free-text instructions added to the prompt.
			{Name: "prompt", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: block},
			// The model answering, DefaultModel when absent.
			{Name: "model", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: block},
			// The effort the model spends, DefaultEffort when absent.
			{Name: "effort", MinArgs: 1, MaxArgs: 1, ArgKinds: str, Placements: block, Values: Efforts},
		},
	}
}

// generator is one request's state.
type generator struct {
	*emit.Session

	// block is the target block's position, where a problem with the
	// whole target is reported.
	block *ir.Position

	files map[string][]byte
}

// Generate runs one conversation and returns the files the model wrote.
func (b Backend) Generate(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	g := &generator{Session: emit.NewSession(req, "yolo"), files: map[string][]byte{}}
	for _, t := range g.Model.GetTargets() {
		if t.GetMeta().GetName() == g.Target {
			g.block = t.GetMeta().GetPosition()
		}
	}

	language, ok := g.blockText("language")
	if !ok {
		g.Error(g.block, "a yolo target block needs language(...) naming the language to write")
		return g.Response(nil), nil
	}
	instructions, _ := g.blockText("prompt")
	model, ok := g.blockText("model")
	if !ok {
		model = DefaultModel
	}
	effort, ok := g.blockText("effort")
	if !ok {
		effort = DefaultEffort
	}

	provider := b.Provider
	if provider == nil {
		p, err := FromEnv()
		if err != nil {
			g.Error(g.block, "%v", err)
			return g.Response(nil), nil
		}
		provider = p
	}

	src := req.GetAnalysis().GetSource()
	if src == nil {
		src = analysis.ModelSource(g.Model, g.Target)
	}
	conv := &Conversation{
		Model:    model,
		Effort:   effort,
		System:   system(),
		Tools:    tools,
		Messages: []Message{{Role: User, Text: userPrompt(language, instructions, src)}},
	}
	if err := g.converse(ctx, provider, conv); err != nil {
		g.Error(g.block, "%v", err)
		return g.Response(nil), nil
	}
	if len(g.files) == 0 {
		g.Error(g.block, "the model wrote no files")
		return g.Response(nil), nil
	}
	// The prompt holds an import's line but not what the dependency's own
	// target block generates, so the model guessed each reference to it.
	for _, ext := range g.Model.GetExterns() {
		g.Lossy(emit.LossUnsupported, ext.GetPosition(),
			"%s.%s comes from another file, whose output the model was not shown; check how the files refer to it",
			ext.GetPackage(), ext.GetName())
	}

	paths := make([]string, 0, len(g.files))
	for p := range g.files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	files := make([]*plugin.File, len(paths))
	for i, p := range paths {
		files[i] = &plugin.File{Path: p, Content: g.files[p]}
	}
	return g.Response(files), nil
}

// blockText reads a directive on the target block.
func (g *generator) blockText(name string) (string, bool) {
	d, ok := g.Block(name)
	if !ok {
		return "", false
	}
	return d.GetArgs()[0].GetText(), true
}

// converse sends turns until the model ends one without calling a tool,
// answering each call. An unsafe path ends it with an error.
func (g *generator) converse(ctx context.Context, p Provider, conv *Conversation) error {
	for range maxTurns {
		reply, err := p.Send(ctx, conv)
		if err != nil {
			return fmt.Errorf("asking %s: %w", conv.Model, err)
		}
		reply.Role = Assistant
		conv.Messages = append(conv.Messages, reply)

		switch reply.Stop {
		case StopRefused:
			return fmt.Errorf("%s declined to answer", conv.Model)
		case StopLength:
			return fmt.Errorf("%s's answer was cut off at the token limit", conv.Model)
		}
		if len(reply.Calls) == 0 {
			return nil
		}

		results := make([]Result, 0, len(reply.Calls))
		for _, c := range reply.Calls {
			r, err := g.call(c)
			if err != nil {
				return err
			}
			results = append(results, r)
		}
		conv.Messages = append(conv.Messages, Message{Role: User, Results: results})
	}
	return fmt.Errorf("%s was still calling tools after %d turns", conv.Model, maxTurns)
}

// call answers one tool call. A call the model can correct is refused back
// to it; an unsafe path is an error, since a model that tries one is not
// to be trusted with the rest.
func (g *generator) call(c Call) (Result, error) {
	refuse := func(format string, args ...any) (Result, error) {
		return Result{ID: c.ID, Content: fmt.Sprintf(format, args...), Error: true}, nil
	}
	switch c.Name {
	case writeFile:
		var in struct {
			Path    *string `json:"path"`
			Content *string `json:"content"`
		}
		if err := json.Unmarshal(c.Input, &in); err != nil || in.Path == nil || in.Content == nil {
			return refuse("write_file takes a path and a content, both strings")
		}
		if !SafePath(*in.Path) {
			return Result{}, fmt.Errorf("the model wrote %q, outside the output directory", *in.Path)
		}
		// Cleaned, so two spellings of one file are one file.
		clean := path.Clean(*in.Path)
		if clean == "." {
			return refuse("%q names the output directory, not a file", *in.Path)
		}
		// Compared without case, since the output may land on a filesystem
		// that ignores it, as macOS's does by default.
		fold := strings.ToLower(clean)
		for p := range g.files {
			lp := strings.ToLower(p)
			if lp == fold && p != clean {
				return refuse("%s and %s differ only in case, so they are one file where case is ignored", clean, p)
			}
			if strings.HasPrefix(lp, fold+"/") || strings.HasPrefix(fold, lp+"/") {
				return refuse("%s and %s cannot both be files, since one would be the other's directory", clean, p)
			}
		}
		g.files[clean] = []byte(*in.Content)
		return Result{ID: c.ID, Content: "wrote " + clean}, nil

	case reportLoss:
		var in struct {
			Decl    *string `json:"decl"`
			Code    string  `json:"code"`
			Message *string `json:"message"`
		}
		if err := json.Unmarshal(c.Input, &in); err != nil || in.Decl == nil || in.Message == nil || *in.Message == "" {
			return refuse("report_loss takes a decl, a code, and a message, all strings")
		}
		if !slices.Contains(emit.LossCodes, in.Code) {
			return refuse("%q is not a loss code; use one of %s", in.Code, strings.Join(emit.LossCodes, ", "))
		}
		pos, ok := g.position(*in.Decl)
		if !ok {
			return refuse("the model declares nothing named %q", *in.Decl)
		}
		g.Lossy(in.Code, pos, "%s", *in.Message)
		return Result{ID: c.ID, Content: "reported"}, nil
	}
	return refuse("there is no tool named %q", c.Name)
}

// position finds what report_loss names: a declaration, a declaration's
// field, an enum's variant or a variant's field, or "" for the target
// block.
func (g *generator) position(name string) (*ir.Position, bool) {
	if name == "" {
		return g.block, true
	}
	parts := strings.Split(name, ".")
	for _, d := range g.Own() {
		if d.GetMeta().GetName() != parts[0] {
			continue
		}
		switch len(parts) {
		case 1:
			return d.GetMeta().GetPosition(), true
		case 2:
			for _, f := range d.Fields() {
				if f.GetMeta().GetName() == parts[1] {
					return f.GetMeta().GetPosition(), true
				}
			}
			for _, v := range d.GetEnumeration().GetVariants() {
				if v.GetMeta().GetName() == parts[1] {
					return v.GetMeta().GetPosition(), true
				}
			}
		case 3:
			for _, v := range d.GetEnumeration().GetVariants() {
				if v.GetMeta().GetName() != parts[1] {
					continue
				}
				for _, f := range v.GetFields() {
					if f.GetMeta().GetName() == parts[2] {
						return f.GetMeta().GetPosition(), true
					}
				}
			}
		}
	}
	return nil, false
}

// SafePath reports whether a path keeps to the protocol's rule: relative,
// and with no ".." segment.
func SafePath(p string) bool {
	return filepath.IsLocal(p) && !strings.Contains(p, `\`) &&
		!slices.Contains(strings.Split(p, "/"), "..")
}
