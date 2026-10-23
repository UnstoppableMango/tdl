package protobuf

import (
	"github.com/bufbuild/protocompile/experimental/ast/printer"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
)

// format lays a generated file out with protocompile's printer under its
// Legacy preset, which reproduces buf format, so formatting the output
// changes nothing. A file that does not parse is returned as written, for
// the compiler to report.
func format(path, src string) string {
	file, ok := parser.Parse(path, source.NewFile(path, src), new(report.Report))
	if !ok {
		return src
	}
	out, err := printer.PrintFile(printer.Options{Format: true, Formatting: printer.Legacy()}, file)
	if err != nil {
		return src
	}
	return out
}
