package lsp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/prelude"
)

// Hover describes the name under the cursor: the declaration it refers to
// in canonical form, its deprecation, and its doc comment. A reference is
// answered from the index Definition reads, including one into the
// prelude; a declaration's own name is found in the tree. On the name of a
// declaration, field, or variant where it is written, the hover also lists
// the directives each target block resolves onto it.
func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc := s.store.get(params.TextDocument.URI.FsPath())
	if doc == nil {
		return nil, nil
	}

	snap := s.store.analyze(doc, s.opts...)
	if ref, ok := snap.refs.At(doc.path, snap.index.byteOffset(params.Position)); ok {
		if ref.Target.Filename == "" {
			return nil, nil // unresolved, or qualified into a dependency
		}
		rng := snap.index.span(ref.Pos.Offset, ref.Len)
		if decl := s.declAt(ref.Target); decl != nil {
			return hover(describe(decl), rng), nil
		}
		// A type parameter starts no declaration and has no ref.Decl.
		if ref.Decl == nil {
			return hover("```tdl\n"+ref.Name+"\n```\n\nType parameter.", rng), nil
		}
		return nil, nil
	}

	if snap.file == nil {
		return nil, nil
	}
	for _, decl := range snap.file.Decls {
		if _, ok := decl.(*ast.TargetDecl); ok {
			continue // names a backend, which declares nothing here
		}
		node := declOf(snap.model, decl.Pos())
		rng, ok := snap.index.nameSpan(decl.Pos(), decl.Head().N)
		if ok && within(rng, params.Position) {
			return hover(describe(decl)+targets(node.GetDirectives()), rng), nil
		}
		for _, m := range membersOf(decl, node) {
			rng, ok := snap.index.nameSpan(m.head.P, m.head.N)
			if ok && within(rng, params.Position) {
				return hover(m.describe()+targets(m.directives), rng), nil
			}
		}
	}
	return nil, nil
}

// member is a field or variant a declaration writes, with the target
// directives lowering resolved onto it.
type member struct {
	head       *ast.DeclHead
	field      *ast.Field // nil for a variant
	directives []*ir.Directive
}

// describe renders a member as markdown, as [describe] does a declaration.
func (m member) describe() string {
	var b strings.Builder
	b.WriteString("```tdl\n")
	if m.field != nil {
		b.WriteString(ast.PrintField(m.field))
	} else {
		b.WriteString(m.head.N)
	}
	b.WriteString("\n```\n")
	if len(m.head.Doc) > 0 {
		b.WriteString("\n" + strings.Join(m.head.Doc, "\n") + "\n")
	}
	return b.String()
}

// membersOf lists the fields and variants decl writes, each with the
// directives its lowered node carries. node is nil when the file did not
// lower, and then no member has any.
func membersOf(decl ast.Decl, node *ir.Decl) []member {
	fieldsOf := func(written []ast.Member, lowered []*ir.Field) []member {
		byName := map[string]*ir.Field{}
		for _, f := range lowered {
			byName[f.GetMeta().GetName()] = f
		}
		var out []member
		for _, m := range written {
			if f, ok := m.(*ast.Field); ok {
				out = append(out, member{head: &f.DeclHead, field: f, directives: byName[f.N].GetDirectives()})
			}
		}
		return out
	}

	switch d := decl.(type) {
	case *ast.StructDecl:
		return fieldsOf(d.Members, node.Fields())
	case *ast.ClassDecl:
		return fieldsOf(d.Members, node.GetClass().GetFields())
	case *ast.EnumDecl:
		byName := map[string]*ir.Variant{}
		for _, v := range node.GetEnumeration().GetVariants() {
			byName[v.GetMeta().GetName()] = v
		}
		var out []member
		for _, v := range d.Variants {
			lowered := byName[v.N]
			out = append(out, member{head: &v.DeclHead, directives: lowered.GetDirectives()})
			fields := make([]ast.Member, len(v.Fields))
			for i, f := range v.Fields {
				fields[i] = f
			}
			out = append(out, fieldsOf(fields, lowered.GetFields())...)
		}
		return out
	}
	return nil
}

// declOf is the lowered declaration written at pos, or nil.
func declOf(model *ir.Model, pos ast.Position) *ir.Decl {
	for _, d := range model.GetDecls() {
		p := d.GetMeta().GetPosition()
		if p.GetFilename() == pos.Filename && int(p.GetLine()) == pos.Line && int(p.GetColumn()) == pos.Col {
			return d
		}
	}
	return nil
}

// targets renders the directives every target block resolves onto a
// node, one line per target in the order the blocks first name it, so a
// field reads with its number where it is declared.
func targets(directives []*ir.Directive) string {
	if len(directives) == 0 {
		return ""
	}
	var order []string
	calls := map[string][]string{}
	for _, d := range directives {
		t := d.GetTarget()
		if _, ok := calls[t]; !ok {
			order = append(order, t)
		}
		call := "`" + d.Call() + "`"
		if from := d.GetFromClass(); from != nil {
			call += " (via " + from.GetName() + ")"
		}
		calls[t] = append(calls[t], call)
	}

	var b strings.Builder
	b.WriteString("\n---\n\n")
	for _, t := range order {
		b.WriteString("**" + t + "**: " + strings.Join(calls[t], ", ") + "  \n")
	}
	return b.String()
}

// describe renders a declaration as markdown: its canonical form, then its
// deprecation, then its doc comment.
func describe(decl ast.Decl) string {
	var b strings.Builder
	b.WriteString("```tdl\n")
	b.WriteString(ast.PrintDecl(decl))
	b.WriteString("```\n")

	head := decl.Head()
	if head.Dep != nil {
		b.WriteString("\n**Deprecated**")
		if head.Dep.Reason != "" {
			b.WriteString(": " + head.Dep.Reason)
		}
		b.WriteString("\n")
	}
	if len(head.Doc) > 0 {
		b.WriteString("\n" + strings.Join(head.Doc, "\n") + "\n")
	}
	return b.String()
}

func hover(markdown string, rng protocol.Range) *protocol.Hover {
	return &protocol.Hover{
		Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: markdown},
		Range:    &rng,
	}
}

// declAt is the declaration that starts at pos.
func (s *Server) declAt(pos ast.Position) ast.Decl {
	file := s.parsed(pos.Filename)
	if file == nil {
		return nil
	}
	for _, decl := range file.Decls {
		if p := decl.Pos(); p.Line == pos.Line && p.Col == pos.Col {
			return decl
		}
	}
	return nil
}

// parsed is the tree of the file a reference points into: the editor's
// text when open, the disk's when not, and the embedded source for the
// prelude. A file that fails to parse yields what the parser recovered.
func (s *Server) parsed(path string) *ast.File {
	if path == prelude.Name {
		return preludeFile()
	}
	if doc := s.store.get(path); doc != nil {
		return s.store.analyze(doc, s.opts...).file
	}
	if !filepath.IsAbs(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	file, _ := parser.Parse(path, strings.NewReader(string(data)))
	return file
}

// preludeFile is the embedded prelude's tree, parsed once.
var preludeFile = sync.OnceValue(func() *ast.File {
	file, _ := parser.Parse(prelude.Name, strings.NewReader(prelude.Source))
	return file
})

// within reports whether a position falls inside a range, end included,
// so a cursor just past a name is still on it.
func within(rng protocol.Range, pos protocol.Position) bool {
	before := func(a, b protocol.Position) bool {
		return a.Line < b.Line || a.Line == b.Line && a.Character <= b.Character
	}
	return before(rng.Start, pos) && before(pos, rng.End)
}
