package analysis

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// ImportedName is the filename positions in an import response's source
// carry.
const ImportedName = "<imported>"

// Imported is the model an import response holds: its model, or else its
// source, parsed and lowered. It is nil when the response holds neither.
func Imported(resp *plugin.ImportResponse) (*ir.Model, error) {
	if m := resp.GetModel(); m != nil {
		return m, nil
	}
	src := resp.GetSource()
	if src == "" {
		return nil, nil
	}
	file, err := parser.Parse(ImportedName, strings.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("the source the backend returned does not parse: %w", err)
	}
	m, diags := sema.Lower(file)
	if len(diags) > 0 {
		return nil, fmt.Errorf("the source the backend returned does not lower: %w", diags)
	}
	return m, nil
}
