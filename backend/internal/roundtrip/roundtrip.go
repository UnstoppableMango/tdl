// Package roundtrip runs a backend forward and back over the corpus in
// testdata/roundtrip, holding it to docs/design/reverse.md's goal.
//
// A case is a directory. One holding source.tdl is model first: the model
// is generated with the `roundtrip` directive on and must import back
// equal, positions aside, and generated with it off must warn exactly the
// loss codes in lossy.golden. One holding expected.tdl is schema first:
// its other files must import to expected.tdl, warning exactly the codes in
// lossy.golden, and expected.tdl must regenerate them, equal under the
// target's normal form.
package roundtrip

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

var update = flag.Bool("update", false, "rewrite lossy.golden and expected.tdl from what the backend does")

// The files a case directory holds besides the target's own.
const (
	Source   = "source.tdl"
	Expected = "expected.tdl"
	Golden   = "lossy.golden"
)

// Directive is the bare target-block directive that makes a backend write
// every fact its mapping would lose.
const Directive = "roundtrip"

// Target is one backend and how its files compare.
type Target struct {
	Backend plugin.Backend

	// Normalize puts files into the target's normal form, so two sets of
	// files that mean the same compare equal. It is given every file at
	// once, since one may import another, and returns them by the same
	// paths. Nil compares bytes.
	Normalize func(files []*plugin.File) ([]*plugin.File, error)

	// Needs says why the target's reader cannot run here, such as a tool
	// missing from PATH, which skips its cases. Nil needs nothing.
	Needs func() error
}

// Run runs every case in dir, a target's directory of the corpus, and
// then each model-first source in smoke. A missing dir is an empty corpus.
// A backend that does not import may have no cases, and skips smoke.
func Run(t *testing.T, target Target, dir string, smoke ...string) {
	t.Helper()
	if target.Needs != nil {
		if err := target.Needs(); err != nil {
			t.Skip(err)
		}
	}

	cases, err := cases(dir)
	if err != nil {
		t.Fatal(err)
	}
	importer, imports := asImporter(target.Backend)
	name := target.Backend.Describe().Name
	if !imports && len(cases) > 0 {
		t.Fatalf("%s has %d round-trip case(s) and does not import", name, len(cases))
	}

	for _, c := range cases {
		t.Run(filepath.Base(c), func(t *testing.T) {
			runCase(t, target, importer, c)
		})
	}

	for _, path := range smoke {
		t.Run("smoke", func(t *testing.T) {
			if !imports {
				t.Skipf("%s does not import yet", name)
			}
			model := lower(t, path, read(t, path))
			comesBack(t, target, importer, path, model)
		})
	}
}

func asImporter(b plugin.Backend) (plugin.Importer, bool) {
	i, ok := b.(plugin.Importer)
	return i, ok && b.Describe().Reverse
}

// cases lists the case directories under dir, sorted.
func cases(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func runCase(t *testing.T, target Target, importer plugin.Importer, dir string) {
	source, expected := filepath.Join(dir, Source), filepath.Join(dir, Expected)
	hasSource, hasExpected := exists(source), exists(expected)
	switch {
	case hasSource && hasExpected:
		t.Fatalf("%s holds both %s and %s; a case runs one direction", dir, Source, Expected)
	case hasSource:
		modelFirst(t, target, importer, source)
	case hasExpected:
		schemaFirst(t, target, importer, dir)
	default:
		t.Fatalf("%s holds neither %s nor %s", dir, Source, Expected)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// modelFirst generates source.tdl without annotations, checking the loss
// codes, and with them, checking the model comes back.
func modelFirst(t *testing.T, target Target, importer plugin.Importer, source string) {
	model := lower(t, source, read(t, source))

	resp := generate(t, target, model)
	golden(t, filepath.Join(filepath.Dir(source), Golden), resp.GetDiagnostics())

	comesBack(t, target, importer, source, model)
}

// comesBack generates model with the roundtrip directive on, which must
// warn about nothing, and imports the files, which must print to TDL
// lowering to the same model.
func comesBack(t *testing.T, target Target, importer plugin.Importer, source string, model *ir.Model) {
	t.Helper()
	name := target.Backend.Describe().Name

	annotated := withDirective(model, name)
	resp := generate(t, target, annotated)
	// The annotations carry every fact, so nothing is lost to warn about.
	for _, d := range resp.GetDiagnostics() {
		t.Errorf("generating %s with %s warns: %s", source, Directive, d.GetMessage())
	}

	imported := importFiles(t, importer, name, resp.GetFiles())
	printed := ast.Fprint(unlower.File(imported.GetModel()))
	got := lower(t, source, printed)

	want, have := withoutDirective(model, name, model), withoutDirective(got, name, model)
	if a, b := ir.WithoutPositions(want), ir.WithoutPositions(have); !proto.Equal(a, b) {
		t.Errorf("%s does not come back from %s\n--- imported ---\n%s--- got ---\n%s--- want ---\n%s",
			source, name, printed, ir.Dump(b), ir.Dump(a))
	}
}

// schemaFirst imports a case's target files, which must print to
// expected.tdl, and regenerates them from it.
func schemaFirst(t *testing.T, target Target, importer plugin.Importer, dir string) {
	name := target.Backend.Describe().Name
	files := targetFiles(t, dir)
	if len(files) == 0 {
		t.Fatalf("%s holds no %s files to import", dir, name)
	}

	resp := importFiles(t, importer, name, files)
	golden(t, filepath.Join(dir, Golden), resp.GetDiagnostics())

	expected := filepath.Join(dir, Expected)
	printed := ast.Fprint(unlower.File(resp.GetModel()))
	if *update {
		write(t, expected, printed)
	}
	if want := read(t, expected); printed != want {
		t.Errorf("%s imports to\n%s\nwant %s:\n%s", dir, printed, Expected, want)
	}

	regenerated := generate(t, target, lower(t, expected, read(t, expected)))
	same(t, target, files, regenerated.GetFiles())
}

// targetFiles reads every file in a case but the harness's own, with paths
// relative to the case.
func targetFiles(t *testing.T, dir string) []*plugin.File {
	t.Helper()
	var files []*plugin.File
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == Expected || rel == Golden {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, &plugin.File{Path: filepath.ToSlash(rel), Content: data})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// same compares regenerated files with the originals under the target's
// normal form.
func same(t *testing.T, target Target, want, got []*plugin.File) {
	t.Helper()
	index := func(files []*plugin.File) map[string][]byte {
		m := map[string][]byte{}
		for _, f := range normal(t, target, files) {
			m[f.GetPath()] = f.GetContent()
		}
		return m
	}
	w, g := index(want), index(got)
	if a, b := slices.Sorted(maps.Keys(w)), slices.Sorted(maps.Keys(g)); !slices.Equal(a, b) {
		t.Fatalf("regenerated %v, want %v", b, a)
	}
	for path, content := range w {
		if string(g[path]) != string(content) {
			t.Errorf("%s regenerates differently\n--- got ---\n%s\n--- want ---\n%s", path, g[path], content)
		}
	}
}

func normal(t *testing.T, target Target, files []*plugin.File) []*plugin.File {
	t.Helper()
	if target.Normalize == nil {
		return files
	}
	out, err := target.Normalize(files)
	if err != nil {
		t.Fatalf("normalizing: %v", err)
	}
	return out
}

func generate(t *testing.T, target Target, model *ir.Model) *plugin.Response {
	t.Helper()
	name := target.Backend.Describe().Name
	resp, err := target.Backend.Generate(context.Background(), &plugin.Request{Target: name, Model: model, Out: "out"})
	if err != nil {
		t.Fatalf("generating %s: %v", name, err)
	}
	fatal(t, "generating", resp.GetDiagnostics())
	return resp
}

func importFiles(t *testing.T, importer plugin.Importer, name string, files []*plugin.File) *plugin.ImportResponse {
	t.Helper()
	resp, err := importer.Import(context.Background(), &plugin.ImportRequest{Target: name, Files: files})
	if err != nil {
		t.Fatalf("importing %s: %v", name, err)
	}
	fatal(t, "importing", resp.GetDiagnostics())
	if resp.GetModel() == nil {
		t.Fatalf("%s imported no model", name)
	}
	return resp
}

func fatal(t *testing.T, doing string, diags []*plugin.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
			t.Fatalf("%s: %s:%d:%d: %s", doing, d.GetPosition().GetFilename(), d.GetPosition().GetLine(), d.GetPosition().GetColumn(), d.GetMessage())
		}
	}
}

// golden checks the loss codes diagnostics carry against a golden file of
// one code per line, sorted. A missing file lists none.
func golden(t *testing.T, path string, diags []*plugin.Diagnostic) {
	t.Helper()
	var codes []string
	for _, d := range diags {
		if c := d.GetCode(); c != "" {
			codes = append(codes, c)
		}
	}
	slices.Sort(codes)
	codes = slices.Compact(codes)

	if *update {
		if len(codes) == 0 {
			_ = os.Remove(path)
		} else {
			write(t, path, strings.Join(codes, "\n")+"\n")
		}
	}

	var want []string
	if data, err := os.ReadFile(path); err == nil {
		want = strings.Fields(string(data))
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if !slices.Equal(codes, want) {
		t.Errorf("loss codes %v, want %v from %s", codes, want, path)
	}
}

// withDirective returns model with a bare roundtrip directive on its block
// for target, adding a block when there is none.
func withDirective(model *ir.Model, target string) *ir.Model {
	out := proto.Clone(model).(*ir.Model)
	d := &ir.Directive{Name: Directive, Target: target}
	for _, b := range out.GetTargets() {
		if b.GetMeta().GetName() == target {
			b.Directives = append(b.Directives, d)
			return out
		}
	}
	out.Targets = append(out.Targets, &ir.TargetBlock{
		Meta:       &ir.Meta{Name: target, Order: int32(len(out.GetTargets()))},
		ForPackage: out.GetPackage(),
		Directives: []*ir.Directive{d},
	})
	return out
}

// withoutDirective drops the roundtrip directive from model's blocks for
// target, and a block for target left empty when original had none, so a
// backend may write the directive back or not.
func withoutDirective(model *ir.Model, target string, original *ir.Model) *ir.Model {
	hadBlock := slices.ContainsFunc(original.GetTargets(), func(b *ir.TargetBlock) bool {
		return b.GetMeta().GetName() == target
	})

	out := proto.Clone(model).(*ir.Model)
	out.Targets = slices.DeleteFunc(out.Targets, func(b *ir.TargetBlock) bool {
		if b.GetMeta().GetName() != target {
			return false
		}
		b.Directives = slices.DeleteFunc(b.Directives, func(d *ir.Directive) bool {
			return d.GetName() == Directive && len(d.GetArgs()) == 0
		})
		return !hadBlock && len(b.GetDirectives()) == 0
	})
	return out
}

func lower(t *testing.T, path, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing:\n%s\n%v", src, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatalf("lowering:\n%s\n%v", src, diags)
	}
	return model
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
