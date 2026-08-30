package gen

import (
	"context"
	"fmt"

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
// directories read from each block's `out` directive.
//
// The directive is where workflow.md puts it: a target block declares
// where its output goes, and the command line overrides it.
func Targets(model *ir.Model, override string) ([]Target, error) {
	var targets []Target
	for _, block := range model.GetTargets() {
		out := override
		if out == "" {
			out = outOf(block)
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

// outOf reads a block's `out("...")` directive.
func outOf(block *ir.TargetBlock) string {
	for _, d := range block.GetDirectives() {
		if d.GetName() != "out" || len(d.GetArgs()) == 0 {
			continue
		}
		return d.GetArgs()[0].GetText()
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
	Target      string
	Written     []string
	Removed     []string
	Stale       []Stale
	Diagnostics []*plugin.Diagnostic
}

// Run generates one target and does what mode says with the result.
func Run(ctx context.Context, backend plugin.Backend, target Target, model *ir.Model, mode Mode) (Result, error) {
	resp, err := backend.Generate(ctx, &plugin.Request{
		Target: target.Name,
		Model:  model,
		Out:    target.Out,
		DryRun: mode == ModeVerify,
	})
	if err != nil {
		return Result{Target: target.Name}, fmt.Errorf("target %s: %w", target.Name, err)
	}

	result := Result{Target: target.Name, Diagnostics: resp.GetDiagnostics()}
	if fatal(resp.GetDiagnostics()) {
		return result, fmt.Errorf("target %s reported errors", target.Name)
	}

	if mode == ModeVerify {
		stale, err := Verify(target.Out, resp.GetFiles())
		result.Stale = stale
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

	// The marker goes down before the files, so a directory tdl wrote is
	// recognisable even if writing fails part way through.
	if err := Mark(target.Out); err != nil {
		return result, fmt.Errorf("target %s: %w", target.Name, err)
	}

	written, err := Write(target.Out, resp.GetFiles())
	result.Written = written
	if err != nil {
		return result, fmt.Errorf("target %s: %w", target.Name, err)
	}
	return result, nil
}

// fatal reports whether any diagnostic is an error rather than a warning.
func fatal(diags []*plugin.Diagnostic) bool {
	for _, d := range diags {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
			return true
		}
	}
	return false
}
