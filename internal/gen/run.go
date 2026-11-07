package gen

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Target is one target block resolved into what a backend needs.
type Target struct {
	Name  string
	Out   string
	Block *ir.TargetBlock
}

// Targets returns the target blocks in a model, with their output
// directories read from each block's `out` directive unless override is
// set. A relative directive is relative to the file declaring the block,
// as an import or an include is.
func Targets(model *ir.Model, override string) ([]Target, error) {
	var targets []Target
	for _, block := range model.GetTargets() {
		out := override
		if out == "" {
			out = Out(block)
		}
		if out == "" {
			return nil, fmt.Errorf("target %s has no out directive and no -o was given", block.GetMeta().GetName())
		}
		targets = append(targets, Target{
			Name:  block.GetMeta().GetName(),
			Out:   out,
			Block: block,
		})
	}
	return targets, nil
}

// Out reads a block's `out("...")` directive, relative to the file
// declaring the block, or returns "" when the block has none.
func Out(block *ir.TargetBlock) string {
	for _, d := range block.GetDirectives() {
		if d.GetName() != host.OutDirective || len(d.GetArgs()) == 0 {
			continue
		}
		out := d.GetArgs()[0].GetText()
		if file := block.GetMeta().GetPosition().GetFilename(); out != "" && file != "" && !filepath.IsAbs(out) {
			out = filepath.Join(filepath.Dir(file), out)
		}
		return out
	}
	return ""
}

// Mode is what a run does with the files a backend returns.
type Mode int

const (
	// ModeWrite writes them.
	ModeWrite Mode = iota
	// ModeVerify compares them against disk and writes nothing.
	ModeVerify
	// ModeClean empties the output directory first, then writes.
	ModeClean
)

// Result is what one target produced.
type Result struct {
	Target   string
	Written  []string
	Removed  []string
	Stale    []Stale
	Expected []string // what a verify run would write
	Kept     []string // what a nondeterministic backend wrote before, kept

	Diagnostics []*plugin.Diagnostic
}

// Run generates one target and does what mode says with the result.
func Run(ctx context.Context, backend plugin.Backend, target Target, model *ir.Model, mode Mode) (Result, error) {
	req, problems := analysis.Request(model, backend.Describe(), target.Name, target.Out)
	if host.Fatal(problems) {
		return Result{Target: target.Name, Diagnostics: problems}, fmt.Errorf("target %s cannot be generated from this model", target.Name)
	}
	req.DryRun = mode == ModeVerify

	var hash string
	if backend.Describe().Nondeterministic {
		var err error
		if hash, err = RequestHash(req); err != nil {
			return Result{Target: target.Name}, fmt.Errorf("target %s: %w", target.Name, err)
		}
		if result, done, err := reuse(target, hash, mode); done || err != nil {
			return result, err
		}
	}

	resp, err := backend.Generate(ctx, req)
	if err != nil {
		return Result{Target: target.Name}, fmt.Errorf("target %s: %w", target.Name, err)
	}

	result := Result{Target: target.Name, Diagnostics: resp.GetDiagnostics()}
	if host.Fatal(resp.GetDiagnostics()) {
		return result, fmt.Errorf("target %s reported errors", target.Name)
	}

	if mode == ModeVerify {
		stale, expected, err := Verify(target.Out, resp.GetFiles())
		result.Stale, result.Expected = stale, expected
		if err != nil {
			return result, fmt.Errorf("target %s: %w", target.Name, err)
		}
		return result, nil
	}

	if mode == ModeClean {
		removed, err := Clean(target.Out)
		result.Removed = removed
		if err != nil {
			return result, fmt.Errorf("target %s: %w", target.Name, err)
		}
	}

	// Written files are listed even when writing fails part way.
	written, err := Write(target.Out, resp.GetFiles())
	result.Written = written
	err = errors.Join(err, Mark(target.Out, written))
	if hash != "" && err == nil {
		err = Keep(target.Out, target.Name, hash, written, resp.GetDiagnostics())
	}
	if err != nil {
		return result, fmt.Errorf("target %s: %w", target.Name, err)
	}
	return result, nil
}

// reuse decides, for a nondeterministic backend, whether the files it wrote
// for the same request still stand, so the request need not be sent. A
// verify run never sends it: a backend that answers differently each time
// would always differ, so a changed request is stale without asking.
// A clean run always regenerates.
func reuse(target Target, hash string, mode Mode) (Result, bool, error) {
	result := Result{Target: target.Name}
	if mode == ModeClean {
		return result, false, nil
	}
	kept, ok, err := Recorded(target.Out, target.Name)
	if err != nil {
		return result, true, fmt.Errorf("target %s: %w", target.Name, err)
	}

	same := ok && kept.Complete() && kept.Request == hash
	var stale []Stale
	if same {
		if stale, err = kept.Check(); err != nil {
			return result, true, fmt.Errorf("target %s: %w", target.Name, err)
		}
		// Files kept are reported with the warnings they were written with.
		result.Diagnostics = kept.Diagnostics
	}

	switch {
	case mode == ModeVerify && !ok:
		result.Stale = []Stale{{Path: target.Out, Reason: "target " + target.Name + " has never been generated here"}}
	case mode == ModeVerify && !kept.Complete():
		result.Stale = []Stale{{Path: filepath.Join(target.Out, MarkerName), Reason: "the record of target " + target.Name + "'s files is incomplete"}}
		result.Expected = kept.Paths()
	case mode == ModeVerify && !same:
		result.Stale = []Stale{{Path: target.Out, Reason: "target " + target.Name + "'s request changed since its files were generated"}}
		result.Expected = kept.Paths()
	case mode == ModeVerify:
		result.Stale, result.Expected = stale, kept.Paths()
	case same && len(stale) == 0:
		result.Kept = kept.Paths()
	default:
		// A changed request, an incomplete record, or a kept file edited or
		// removed: regenerate.
		return result, false, nil
	}
	return result, true, nil
}
