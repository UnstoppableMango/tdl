package gen

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

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
			out = outOf(block)
			if file := block.GetMeta().GetPosition().GetFilename(); out != "" && file != "" && !filepath.IsAbs(out) {
				out = filepath.Join(filepath.Dir(file), out)
			}
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
		if d.GetName() != outDirective || len(d.GetArgs()) == 0 {
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
	Target   string
	Written  []string
	Removed  []string
	Stale    []Stale
	Expected []string // what a verify run would write

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
	if Fatal(resp.GetDiagnostics()) {
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
	if err := errors.Join(err, Mark(target.Out, written)); err != nil {
		return result, fmt.Errorf("target %s: %w", target.Name, err)
	}
	return result, nil
}

// Fatal reports whether any diagnostic is an error rather than a warning.
func Fatal(diags []*plugin.Diagnostic) bool {
	return slices.ContainsFunc(diags, func(d *plugin.Diagnostic) bool {
		return d.GetSeverity() == plugin.Severity_SEVERITY_ERROR
	})
}

// Silence drops the warnings whose loss code is allowed. An error is never
// dropped, and neither is a warning with no code.
func Silence(diags []*plugin.Diagnostic, allowed []string) []*plugin.Diagnostic {
	return slices.DeleteFunc(slices.Clone(diags), func(d *plugin.Diagnostic) bool {
		return d.GetSeverity() == plugin.Severity_SEVERITY_WARNING &&
			d.GetCode() != "" && slices.Contains(allowed, d.GetCode())
	})
}
