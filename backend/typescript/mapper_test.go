package typescript_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
)

// mapperSession is a Mapper served over a pipe, as tsc runs it.
type mapperSession struct {
	t    *testing.T
	in   *io.PipeWriter
	out  *bufio.Reader
	next int
	done chan error
}

func serveMapper(t *testing.T) *mapperSession {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := &mapperSession{t: t, in: inW, out: bufio.NewReader(outR), done: make(chan error, 1)}
	go func() {
		err := typescript.Mapper{}.Serve(t.Context(), pipe{inR, outW})
		outW.Close()
		s.done <- err
	}()
	t.Cleanup(func() {
		inW.Close()
		if err := <-s.done; err != nil {
			t.Errorf("Serve: %v", err)
		}
	})
	return s
}

// pipe is the mapper's end of a session: what the test writes, and where
// the mapper answers.
type pipe struct {
	*io.PipeReader
	*io.PipeWriter
}

func (p pipe) Close() error {
	p.PipeReader.Close()
	return p.PipeWriter.Close()
}

// call sends one request and decodes its result into result.
func (s *mapperSession) call(method string, params, result any) {
	s.t.Helper()
	s.next++
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "api" + strconv.Itoa(s.next), "method": method, "params": params})
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := fmt.Fprintf(s.in, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		s.t.Fatal(err)
	}

	length := -1
	for {
		line, err := s.out.ReadString('\n')
		if err != nil {
			s.t.Fatalf("%s: reading the response: %v", method, err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			length, _ = strconv.Atoi(v)
		}
	}
	reply := make([]byte, length)
	if _, err := io.ReadFull(s.out, reply); err != nil {
		s.t.Fatal(err)
	}

	var resp struct {
		ID     string          `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct{ Message string }
	}
	if err := json.Unmarshal(reply, &resp); err != nil {
		s.t.Fatal(err)
	}
	if resp.ID != "api"+strconv.Itoa(s.next) {
		s.t.Errorf("%s: answered %q, want %q", method, resp.ID, "api"+strconv.Itoa(s.next))
	}
	if resp.Error != nil {
		s.t.Fatalf("%s: %s", method, resp.Error.Message)
	}
	if result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s *mapperSession) transform(path string) typescript.Module {
	s.t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		s.t.Fatal(err)
	}
	var mod typescript.Module
	s.call("transform", map[string]any{"fileName": path, "content": string(src), "projectHandle": "p"}, &mod)
	return mod
}

// writeFiles writes each file into a new directory and returns it.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A session runs the protocol from initialize to closeProject, refusing
// options and answering every request by its id.
func TestMapperSession(t *testing.T) {
	s := serveMapper(t)

	var init struct {
		ProtocolVersion  int    `json:"protocolVersion"`
		PositionEncoding string `json:"positionEncoding"`
		DiagnosticSource string `json:"diagnosticSource"`
	}
	s.call("initialize", map[string]any{"positionEncodings": []string{"utf-8", "utf-16"}}, &init)
	if init.ProtocolVersion != 1 || init.PositionEncoding != "utf-8" || init.DiagnosticSource != "tdl" {
		t.Errorf("initialize = %+v", init)
	}

	var open struct {
		OptionDiagnostics []typescript.OptionDiagnostic `json:"optionDiagnostics"`
	}
	s.call("openProject", map[string]any{"projectHandle": "p", "options": map[string]any{"zod": true}}, &open)
	if len(open.OptionDiagnostics) != 1 || open.OptionDiagnostics[0].Path[0] != "zod" || open.OptionDiagnostics[0].Code != typescript.CodeOption {
		t.Errorf("optionDiagnostics = %+v, want one refusing zod", open.OptionDiagnostics)
	}

	dir := writeFiles(t, map[string]string{"shop.tdl": "package shop\n\ntype Order: Entity {\n  id: string\n}\n"})
	mod := s.transform(filepath.Join(dir, "shop.tdl"))
	if mod.Extension != ".mts" || !strings.Contains(mod.Text, "export interface Order {") || len(mod.Diagnostics) != 0 {
		t.Errorf("transform = %+v", mod)
	}

	s.call("closeProject", map[string]any{"projectHandle": "p"}, nil)
}

// A module is the text tdl gen writes, for the smoke model and every
// conformance case the backend generates.
func TestMapperModuleIsGenerated(t *testing.T) {
	corpus := filepath.Join("..", "..", "testdata", "conformance")
	dirs := []string{filepath.Join("..", "..", "testdata", "gen", "smoke")}
	for _, c := range []string{"aliases", "comments", "deprecated", "enum_variants", "mixin_include", "newtype", "value"} {
		dirs = append(dirs, filepath.Join(corpus, c))
	}

	s := serveMapper(t)
	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join(dir, "source.tdl"))
			if err != nil {
				t.Fatal(err)
			}
			s.t = t // a failure belongs to the subtest, not its parent
			mod := s.transform(path)
			for _, d := range mod.Diagnostics {
				t.Errorf("diagnostic tdl%d at %d: %s", d.Code, d.Start, d.MessageText)
			}
			if want := generated(t, path); mod.Text != want {
				t.Errorf("module differs from tdl gen:\n%s\nwant:\n%s", mod.Text, want)
			}
		})
	}
}

// generated is what the backend writes for the file at path.
func generated(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	file, err := parser.Parse(path, f)
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	req, _ := analysis.Request(model, typescript.Backend{}.Describe(), typescript.Name, "")
	resp, err := typescript.Backend{}.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFiles()) == 0 {
		return "export {};\n"
	}
	return string(resp.GetFiles()[0].GetContent())
}

// A declaration from another file is imported from that .tdl file, by the
// path the TDL import names, whether or not the file has a typescript
// block.
func TestMapperImportsTheSourceFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"money.tdl": "package money\n\ntype Money {\n  amount: decimal\n}\n",
		"shop.tdl":  "package shop\n\nimport \"money.tdl\" as money\n\ntype Order: Entity {\n  total: money.Money\n}\n",
	})
	mod := serveMapper(t).transform(filepath.Join(dir, "shop.tdl"))
	if len(mod.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", mod.Diagnostics)
	}
	for _, want := range []string{`import type { Money } from "./money.tdl";`, "total: Money;"} {
		if !strings.Contains(mod.Text, want) {
			t.Errorf("module lacks %q:\n%s", want, mod.Text)
		}
	}
}

// Each kind of problem is reported under its code, at the range of the
// .tdl file it is about, and an error leaves the module empty.
func TestMapperDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		files   map[string]string
		code    int
		at      string // the text the diagnostic starts at
		message string
		module  bool // whether the module still declares types
	}{
		{
			name:    "syntax",
			files:   map[string]string{"a.tdl": "package a\n\ntype Order {\n  id string\n}\n"},
			code:    typescript.CodeSyntax,
			at:      "string\n}",
			message: "expected",
		},
		{
			name:    "lowering",
			files:   map[string]string{"a.tdl": "package a\n\ntype Order {\n  id: Strin\n}\n"},
			code:    typescript.CodeLowering,
			at:      "Strin",
			message: "undefined: Strin",
		},
		{
			name: "dependency",
			files: map[string]string{
				"a.tdl": "package a\n\nimport \"b.tdl\" as b\n\ntype Order {\n  id: b.Id\n}\n",
				"b.tdl": "package b\n\ntype Id string\n",
			},
			code:    typescript.CodeLowering,
			at:      `import "b.tdl"`,
			message: `import "b.tdl" does not parse`,
		},
		{
			name:    "directive",
			files:   map[string]string{"a.tdl": "package a\n\nenum Shape { Dot }\n\ntarget typescript for a {\n  discriminant(1)\n}\n"},
			code:    typescript.CodeDirective,
			at:      "1)",
			message: "discriminant argument 1",
		},
		{
			name:    "model",
			files:   map[string]string{"a.tdl": "package a\n\ntype Code: string where { length(5..2) }\n"},
			code:    typescript.CodeModel,
			at:      "",
			message: "can never hold",
		},
		{
			name:    "skipped",
			files:   map[string]string{"a.tdl": "package a\n\ntype Key { id: string }\ntype Index {\n  byKey: {Key -> string}\n}\ntype Order { id: string }\n"},
			code:    typescript.CodeSkipped,
			at:      "{Key",
			message: "record key",
			module:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			path := filepath.Join(dir, "a.tdl")
			mod := serveMapper(t).transform(path)
			if len(mod.Diagnostics) == 0 {
				t.Fatal("no diagnostics")
			}
			d := mod.Diagnostics[0]
			if d.Code != tc.code || !strings.Contains(d.MessageText, tc.message) {
				t.Errorf("diagnostic = tdl%d %q, want tdl%d containing %q", d.Code, d.MessageText, tc.code, tc.message)
			}
			if tc.at != "" {
				if want := strings.Index(tc.files["a.tdl"], tc.at); d.Start != want {
					t.Errorf("diagnostic starts at %d (%q), want %d (%q)", d.Start, tc.files["a.tdl"][d.Start:], want, tc.at)
				}
			}
			if declares := strings.Contains(mod.Text, "export "+"interface") || strings.Contains(mod.Text, "export type"); declares != tc.module {
				t.Errorf("module declares types: %v, want %v:\n%s", declares, tc.module, mod.Text)
			}
		})
	}
}

// A loss warning is not reported: tsc shows every mapper diagnostic as an
// error, and the loss is tdl gen's to report.
func TestMapperDropsLosses(t *testing.T) {
	dir := writeFiles(t, map[string]string{"a.tdl": "package a\n\n/// An order.\ntype Order: Entity {\n  id: string\n  tags: {string}\n}\n"})
	mod := serveMapper(t).transform(filepath.Join(dir, "a.tdl"))
	if len(mod.Diagnostics) != 0 {
		t.Errorf("diagnostics = %+v, want none", mod.Diagnostics)
	}
}

// tsc 7.1 type checks a TypeScript project importing .tdl files through
// the mapper, and reports a TDL error in the .tdl file.
func TestMapperUnderTsc(t *testing.T) {
	tsc := tsc71(t)

	bin := filepath.Join(t.TempDir(), "tdl-gen-typescript")
	build := exec.Command("go", "build", "-o", bin, "github.com/unstoppablemango/tdl/cmd/tdl-gen-typescript")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	project := t.TempDir()
	if err := os.CopyFS(project, os.DirFS(filepath.Join("testdata", "mapper"))); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(project, "node_modules", "@unmango", "tdl-typescript")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(map[string]any{
		"name":       "@unmango/tdl-typescript",
		"version":    "0.0.0",
		"typescript": map[string]any{"contentMapper": map[string]any{"exec": []string{bin, "content-mapper"}, "compilerOptions": []string{}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "package.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	run := func() (string, error) {
		cmd := exec.Command(tsc, "--noEmit", "--runExternalCode", "-p", project)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run(); err != nil {
		t.Fatalf("tsc rejects the project: %v\n%s", err, out)
	}

	shop := filepath.Join(project, "src", "shop.tdl")
	src, err := os.ReadFile(shop)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shop, []byte(strings.Replace(string(src), "status: Status", "status: Statsu", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run()
	if err == nil || !regexp.MustCompile(`src/shop\.tdl\(\d+,\d+\): error tdl2: undefined: Statsu`).MatchString(out) {
		t.Errorf("tsc = %v, want tdl2 in shop.tdl:\n%s", err, out)
	}
}

// tsc71 is a tsc of 7.1 or later on PATH; without one, or with -short, the
// test skips.
func tsc71(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("-short skips running tsc")
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		t.Skip("tsc is not on PATH")
	}
	out, err := exec.Command(tsc, "--version").Output()
	if err != nil {
		t.Skipf("tsc --version: %v", err)
	}
	m := regexp.MustCompile(`Version (\d+)\.(\d+)`).FindStringSubmatch(string(out))
	if m == nil {
		t.Skipf("tsc --version printed %q", out)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	if major < 7 || (major == 7 && minor < 1) {
		t.Skipf("tsc %s.%s has no content mappers", m[1], m[2])
	}
	return tsc
}

func TestMapperRefusesBadFrames(t *testing.T) {
	for name, frame := range map[string]string{
		"missing":     "Content-Type: x\r\n\r\n{}",
		"negative":    "Content-Length: -1\r\n\r\n",
		"conflicting": "Content-Length: 2\r\nContent-Length: 3\r\n\r\n{}",
	} {
		t.Run(name, func(t *testing.T) {
			if err := (typescript.Mapper{}).Serve(t.Context(), frameConn{strings.NewReader(frame)}); err == nil {
				t.Error("Serve accepted the frame")
			}
		})
	}
}

// frameConn serves one fixed input and discards what the mapper writes.
type frameConn struct{ io.Reader }

func (frameConn) Write(p []byte) (int, error) { return len(p), nil }
func (frameConn) Close() error                { return nil }
