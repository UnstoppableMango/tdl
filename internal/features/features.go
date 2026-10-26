// Package features measures what each backend does with each TDL feature
// and writes the result as markdown tables between marker comments.
//
// A feature is a directory of testdata/features holding source.tdl, a
// small model using it, and a backend's subdirectory may hold its own. Support is read from the backend, not declared:
// generating with no warning is Yes, warning only loss codes is Warns, and
// an error, a skip, or any warning without a code is No. Import generates
// without the roundtrip directive and reads the files back: the same model
// is Yes, a different one Warns with the codes either direction reported,
// and a failure No. What a measurement cannot say, how the target spells a
// feature and why, comes from backend/<name>/features.toml.
package features

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// Source is the file a feature's directory holds. A subdirectory named
// for a backend may hold its own, for a feature that backend reaches only
// through its own directives.
const Source = "source.tdl"

// Placeholder is the target name a feature's source writes its target
// block for, replaced by each backend's name before lowering.
const Placeholder = "feature"

// Row is one feature: the directory of testdata/features holding it, the
// name a table shows, and the loss codes that say the feature itself is
// lost. A source cannot use a feature alone, so a code a row does not own
// is about something else in the source and is not this row's to report.
type Row struct {
	Dir, Title string
	Codes      []string
}

// Rows is every feature, in table order. The loss codes in
// docs/architecture/reverse.md name most of them, spelled out here because
// backend/internal/emit is out of reach, so a Warns cell names a code
// the reader can allow; primitives are left to each design's type table.
var Rows = []Row{
	{"struct-kind", "Entity, value, and mixin", []string{"lossy.struct-kind"}},
	{"include", "Mixin `include`", []string{"lossy.include"}},
	{"enum", "Fieldless enum", nil},
	{"fielded-enum", "Fielded enum", nil},
	{"newtype", "Newtype", []string{"lossy.newtype"}},
	{"alias", "Alias", []string{"lossy.alias"}},
	{"optional", "Optional and nullable", []string{"lossy.optional"}},
	{"collection", "List, set, and map", []string{"lossy.collection"}},
	{"min", "`min`", []string{"lossy.constraint"}},
	{"max", "`max`", []string{"lossy.constraint"}},
	{"length", "`length`", []string{"lossy.constraint"}},
	{"matches", "`matches`", []string{"lossy.constraint"}},
	{"oneOf", "`oneOf`", []string{"lossy.constraint"}},
	{"unique", "`unique`", []string{"lossy.constraint"}},
	{"owned", "`owned`", []string{"lossy.owned"}},
	{"default", "Field default", []string{"lossy.default"}},
	{"key", "`key` directive", []string{"lossy.key"}},
	{"name", "`name` directive", []string{"lossy.name"}},
	{"number", "`number` directive", []string{"lossy.number"}},
	{"doc", "Doc comment", []string{"lossy.doc"}},
	{"generic", "Type parameters", []string{"lossy.generic"}},
	{"class", "Class", []string{"lossy.class"}},
	{"unit", "Unit", []string{"lossy.unit"}},
	{"import", "Import and extern", nil},
}

// Support is what a backend does with a feature.
type Support int

const (
	// Unmeasured is a cell the measurement skipped, such as an import
	// needing a tool missing from PATH.
	Unmeasured Support = iota
	Yes
	Warns
	No
	Planned
)

func (s Support) String() string {
	return [...]string{"Unmeasured", "Yes", "Warns", "No", "Planned"}[s]
}

// Cell is one measurement.
type Cell struct {
	Support Support
	// Codes are the loss codes a Warns cell reported, sorted.
	Codes []string
	// Why is the first diagnostic that made a cell No, for a reader
	// wondering what failed.
	Why string
}

// Measured is one feature through one backend.
type Measured struct {
	Generate, Import Cell
}

// Target is one backend to measure.
type Target struct {
	Name string
	// Backend is nil for a plugin missing from PATH, which leaves every
	// cell Unmeasured.
	Backend plugin.Backend
	// Hand is the target's features.toml, relative to the module root.
	Hand string
	// Needs says why the backend cannot import here, such as a tool
	// missing from PATH, which leaves its Import cells Unmeasured. Nil
	// needs nothing.
	Needs func() error
}

// Measure runs every row in dir, testdata/features, through target.
func Measure(dir string, target Target) (map[string]Measured, error) {
	out := map[string]Measured{}
	if target.Backend == nil {
		for _, row := range Rows {
			out[row.Dir] = Measured{}
		}
		return out, nil
	}
	for _, row := range Rows {
		path := filepath.Join(dir, row.Dir, target.Name, Source)
		if _, err := os.Stat(path); err != nil {
			path = filepath.Join(dir, row.Dir, Source)
		}
		m, err := measure(path, row, target)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", row.Dir, err)
		}
		out[row.Dir] = m
	}
	return out, nil
}

func measure(path string, row Row, target Target) (Measured, error) {
	loader := renaming{target.Name}
	model, err := lower(path, "", loader)
	if err != nil {
		return Measured{}, err
	}
	desc := target.Backend.Describe()

	resp, generated := generate(model, row, target)
	m := Measured{Generate: generated}

	importer, ok := target.Backend.(plugin.Importer)
	switch {
	case !ok || !desc.Reverse:
		m.Import = Cell{Support: No, Why: "does not import"}
	case target.Needs != nil && target.Needs() != nil:
		m.Import = Cell{Support: Unmeasured}
	case generated.Support == No:
		m.Import = Cell{Support: No, Why: "does not generate"}
	default:
		m.Import = readBack(path, row, model, target, importer, resp, loader)
	}
	return m, nil
}

// generate runs the backend the way `tdl gen` does, directive checks
// included, and grades what it reports.
func generate(model *ir.Model, row Row, target Target) (*plugin.Response, Cell) {
	desc := target.Backend.Describe()
	problems := host.CheckDirectives(target.Name, model, desc)
	req, more := analysis.Request(model, desc, target.Name, "out")
	problems = append(problems, more...)
	if host.Fatal(problems) {
		return nil, no(problems)
	}
	resp, err := target.Backend.Generate(context.Background(), req)
	if err != nil {
		return nil, Cell{Support: No, Why: err.Error()}
	}
	return resp, grade(append(problems, resp.GetDiagnostics()...), row)
}

// grade turns diagnostics into a cell: an error or a warning without a
// code is No, the row's own codes are Warns, and anything else is Yes.
func grade(diags []*plugin.Diagnostic, row Row) Cell {
	var codes []string
	for _, d := range diags {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR || d.GetCode() == "" {
			return no(diags)
		}
		if slices.Contains(row.Codes, d.GetCode()) {
			codes = append(codes, d.GetCode())
		}
	}
	if len(codes) == 0 {
		return Cell{Support: Yes}
	}
	slices.Sort(codes)
	return Cell{Support: Warns, Codes: slices.Compact(codes)}
}

func no(diags []*plugin.Diagnostic) Cell {
	c := Cell{Support: No}
	for _, d := range diags {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR || d.GetCode() == "" {
			c.Why = d.GetMessage()
			break
		}
	}
	return c
}

// readBack imports what the backend generated and compares the model it
// prints to with the one generated from. A difference with no loss code
// in either direction is a silent loss, and No; one whose codes the row
// does not own is put down to the rest of the source.
func readBack(path string, row Row, model *ir.Model, target Target, importer plugin.Importer, resp *plugin.Response, loader renaming) Cell {
	imported, err := importer.Import(context.Background(), &plugin.ImportRequest{Target: target.Name, Files: resp.GetFiles()})
	if err != nil {
		return Cell{Support: No, Why: err.Error()}
	}
	if host.Fatal(imported.GetDiagnostics()) {
		return no(imported.GetDiagnostics())
	}
	got, err := analysis.Imported(imported)
	if err != nil {
		return Cell{Support: No, Why: err.Error()}
	}
	if got == nil {
		return Cell{Support: No, Why: "imported no model"}
	}
	relowered, err := lower(path, ast.Fprint(unlower.File(got)), loader)
	if err != nil {
		return Cell{Support: No, Why: err.Error()}
	}

	scoped := analysis.Scope(model, target.Name)
	if proto.Equal(ir.WithoutPositions(scoped), ir.WithoutPositions(relowered)) {
		return Cell{Support: Yes}
	}
	diags := append(slices.Clone(resp.GetDiagnostics()), imported.GetDiagnostics()...)
	if !slices.ContainsFunc(diags, func(d *plugin.Diagnostic) bool { return d.GetCode() != "" }) {
		return Cell{Support: No, Why: "reads back a different model without a loss code"}
	}
	return grade(diags, row)
}

// renaming loads files from disk with their placeholder target blocks
// renamed for a backend, so a feature's dependency can carry one too.
type renaming struct{ target string }

func (r renaming) Load(from, path string) (string, string, error) {
	name, src, err := sema.FSLoader{}.Load(from, path)
	return name, r.rename(src), err
}

func (r renaming) rename(src string) string {
	return strings.ReplaceAll(src, "target "+Placeholder+" for", "target "+r.target+" for")
}

// lower lowers src as the file at path, or the file itself, renamed, when
// src is empty.
func lower(path, src string, loader renaming) (*ir.Model, error) {
	if src == "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		src = loader.rename(string(data))
	}
	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	model, diags := sema.Lower(file, sema.WithLoader(loader))
	if len(diags) > 0 {
		return nil, diags
	}
	return model, nil
}
