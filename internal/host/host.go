// Package host holds what a host does around a backend that does not
// depend on which backends exist: holding a target block's directives to
// the backend's specs, reading which loss codes a project allows, and
// dropping the warnings those codes silence.
//
// internal/gen and every backend that hosts itself, as the typescript
// content mapper does, share it; it imports no backend.
package host

import (
	"slices"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/plugin"
)

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

// Allowed returns the loss codes the nearest tdl.toml above dir allows for
// target.
func Allowed(dir, target string) ([]string, error) {
	cfg, err := config.Find(dir)
	if err != nil {
		return nil, err
	}
	return cfg.Allowed(target), nil
}
