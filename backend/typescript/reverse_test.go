package typescript_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// needsCompiler skips a test where node or tsc is missing, since import
// reads TypeScript with the compiler.
func needsCompiler(t *testing.T) {
	t.Helper()
	if _, _, err := typescript.Compiler(); err != nil {
		t.Skip(err)
	}
}

// importTS imports files by path, and returns the TDL printed from the
// model, with the response's diagnostics.
func importTS(t *testing.T, files map[string]string) (string, []*plugin.Diagnostic) {
	t.Helper()
	needsCompiler(t)
	req := &plugin.ImportRequest{Target: typescript.Name}
	for path, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: path, Content: []byte(src)})
	}
	resp, err := typescript.Backend{}.Import(context.Background(), req)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

func failed(t *testing.T, diags []*plugin.Diagnostic, want string) {
	t.Helper()
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), want) {
		t.Errorf("diagnostics = %v, want one error containing %q", diags, want)
	}
}

func TestImportRefusesWhatItCannotRead(t *testing.T) {
	_, diags := importTS(t, map[string]string{})
	failed(t, diags, "was given 0")

	_, diags = importTS(t, map[string]string{"a.ts": "", "b.ts": ""})
	failed(t, diags, "was given 2")

	_, diags = importTS(t, map[string]string{"a.go": "package a"})
	failed(t, diags, "reads .ts files")

	_, diags = importTS(t, map[string]string{"a.ts": "export interface A {"})
	failed(t, diags, "does not parse")

	_, diags = importTS(t, map[string]string{"a.ts": "/** @tdl source \"type A {\" */\nexport interface A {}\n"})
	failed(t, diags, "is not TDL")
}

// What a handwritten file holds that TDL has no form for, or that
// regenerates differently, warns with its loss code and is read as near as
// TDL comes.
func TestImportWarnsWhatItLoses(t *testing.T) {
	src, diags := importTS(t, map[string]string{"shop.ts": `
/** One. */
/** Two. */
export interface Item {
  id: string;
  readonly count: number;
  /** @see elsewhere */
  note: string;
  run(): void;
  shape: { a: string };
  either: string | number;
  when: Date;
  "1st": string;
}

interface Hidden {
  id: string;
}

export enum Level {
  Low = 1,
}

export class Box {}

export interface Pair<T> {
  first: T;
}

export type Grade = "a" | "b";
`})
	want := map[string]int{
		// count, run, shape, either, when, 1st, Hidden, Level, Box
		emit.LossUnsupported: 9,
		emit.LossDoc:         2, // the two comments, and @see
		emit.LossGeneric:     1, // Pair
	}
	got := map[string]int{}
	for _, d := range diags {
		got[d.GetCode()]++
	}
	for code, n := range want {
		if got[code] != n {
			t.Errorf("%d %s warnings, want %d: %v", got[code], code, n, diags)
		}
	}
	for _, s := range []string{
		"/// Two.\ntype Item {",
		"count: float64",
		"note: string",
		"enum Grade { a b }",
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
}

// model.ts names no package, and any other file name is the package.
func TestImportReadsThePackageFromTheFileName(t *testing.T) {
	for path, want := range map[string]string{"model.ts": "", "shop.ts": "package shop\n"} {
		src, diags := importTS(t, map[string]string{path: "export interface A {\n  b: string;\n}\n"})
		if len(diags) > 0 {
			t.Fatalf("%s: %v", path, diags)
		}
		if got := strings.HasPrefix(src, "package"); got != (want != "") || !strings.HasPrefix(src, want) {
			t.Errorf("%s imports as\n%s", path, src)
		}
	}
}

// Optional and null read back as the option and nullable they are
// generated from, and a property's literals as oneOf.
func TestImportReadsOptionalAndNull(t *testing.T) {
	src, diags := importTS(t, map[string]string{"shop.ts": `export interface A {
  a?: string;
  b: string | null;
  c?: string | null;
  d: (string | null)[];
  e: Record<string, number | null>;
  f: 1 | 2;
}
`})
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	for _, s := range []string{
		"a: string?",
		"b: string | null",
		"c: string? | null",
		"d: [string?]",
		"e: {string -> float64?}",
		"f: float64 where { oneOf(1, 2) }",
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
}

// An annotation's value survives every character a Go string literal
// escapes, and the */ that would end its comment.
func TestAnnotationsSurviveEscapes(t *testing.T) {
	for _, reason := range []string{`a " b`, `ends in \`, `\\"`, "two\nlines", "tab\there", "café", "a */ close"} {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(reason)
		item := "deprecated(\"" + escaped + "\")\ntype A {\n  b: string\n}"
		value := strings.ReplaceAll(strconv.Quote(item), "*/", `*\u002f`)
		src := "/** @tdl source " + value + " */\nexport interface A {\n  b: string;\n}\n"
		got, diags := importTS(t, map[string]string{"shop.ts": src})
		if len(diags) > 0 {
			t.Fatalf("%q: %v", reason, diags)
		}
		if want := `deprecated("` + escaped + `")`; !strings.Contains(got, want) {
			t.Errorf("%q imports as\n%s\nwant %s", reason, got, want)
		}
	}
}

// Annotated output is still a file tsc accepts.
func TestAnnotatedOutputTypeChecks(t *testing.T) {
	for _, path := range []string{
		"../../testdata/gen/smoke/source.tdl",
		"../../testdata/roundtrip/typescript/losses/source.tdl",
		"../../testdata/roundtrip/typescript/docs/source.tdl",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		model, diags := sema.Lower(file)
		if len(diags) > 0 {
			t.Fatal(diags)
		}
		model.Targets = append(model.Targets, &ir.TargetBlock{
			Meta:       &ir.Meta{Name: typescript.Name},
			ForPackage: model.GetPackage(),
			Directives: []*ir.Directive{{Name: "roundtrip", Target: typescript.Name}},
		})
		resp, err := typescript.Backend{}.Generate(context.Background(), &plugin.Request{Target: typescript.Name, Model: model})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.GetDiagnostics()) > 0 {
			t.Errorf("%s: diagnostics = %v", path, resp.GetDiagnostics())
		}
		for _, f := range resp.GetFiles() {
			typecheck(t, f)
		}
	}
}
