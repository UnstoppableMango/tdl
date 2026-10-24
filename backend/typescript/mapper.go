package typescript

// The content mapper lets TypeScript 7.1 import a .tdl file as a module of
// types: tsc spawns `tdl-gen-typescript content-mapper` and asks it, over
// JSON-RPC on standard input and output, for each .tdl file's module.
// docs/design/typescript-content-mapper.md describes the protocol.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
	"go.lsp.dev/jsonrpc2"
)

// protocolVersion is the content mapper protocol this mapper speaks.
const protocolVersion = 1

// diagnosticSource prefixes each diagnostic's code where tsc prints it, as
// in `error tdl2`.
const diagnosticSource = "tdl"

// The code of each kind of diagnostic the mapper reports.
const (
	CodeSyntax    = 1 // the file does not parse
	CodeLowering  = 2 // the file, or a file it imports, does not lower
	CodeDirective = 3 // a typescript directive is used incorrectly
	CodeModel     = 4 // the model cannot be generated for typescript
	CodeSkipped   = 5 // the backend skipped a declaration
	CodeOption    = 6 // the tsconfig entry passes an option
)

// emptyModule is the module of a file that cannot be generated: it names no
// type, so an import of it fails and only the file's own errors say why.
const emptyModule = "export {};\n"

// Mapper serves the content mapper protocol. It reads the files a .tdl
// file imports from disk: tsc sends only the content of the file it asks
// for.
type Mapper struct{}

// Serve answers requests read from rwc until it ends.
func (m Mapper) Serve(ctx context.Context, rwc io.ReadWriteCloser) error {
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(rwc))
	conn.Go(ctx, func(_ context.Context, req *jsonrpc2.Request) (any, error) {
		if !req.IsCall() {
			// No notification needs an answer, and a handler's error on
			// one would end the session.
			return nil, nil
		}
		return m.handle(req.Method(), req.Params())
	})
	<-conn.Done()
	return conn.Err()
}

func (m Mapper) handle(method string, params jsonrpc2.RawMessage) (any, error) {
	switch method {
	case "initialize":
		var p initializeParams
		if err := unmarshal(params, &p); err != nil {
			return nil, err
		}
		if len(p.PositionEncodings) > 0 && !slices.Contains(p.PositionEncodings, "utf-8") {
			return nil, fmt.Errorf("the mapper reports positions in utf-8, which %v leaves out", p.PositionEncodings)
		}
		return initializeResult{ProtocolVersion: protocolVersion, PositionEncoding: "utf-8", DiagnosticSource: diagnosticSource}, nil
	case "openProject":
		var p openProjectParams
		if err := unmarshal(params, &p); err != nil {
			return nil, err
		}
		return openProjectResult{OptionDiagnostics: optionDiagnostics(p.Options)}, nil
	case "transform":
		var p transformParams
		if err := unmarshal(params, &p); err != nil {
			return nil, err
		}
		return m.Transform(p.FileName, p.Content), nil
	case "closeProject":
		return struct{}{}, nil
	}
	return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrMethodNotFound, method)
}

func unmarshal(params jsonrpc2.RawMessage, v any) error {
	if len(params) == 0 {
		return nil
	}
	return json.Unmarshal(params, v)
}

type initializeParams struct {
	PositionEncodings []string `json:"positionEncodings"`
}

type initializeResult struct {
	ProtocolVersion  int    `json:"protocolVersion"`
	PositionEncoding string `json:"positionEncoding"`
	DiagnosticSource string `json:"diagnosticSource"`
}

type openProjectParams struct {
	Options map[string]json.RawMessage `json:"options"`
}

type openProjectResult struct {
	OptionDiagnostics []OptionDiagnostic `json:"optionDiagnostics"`
}

// OptionDiagnostic is a problem with one key of the tsconfig entry's
// options.
type OptionDiagnostic struct {
	Path        []string `json:"path"`
	MessageText string   `json:"messageText"`
	Code        int      `json:"code"`
}

// optionDiagnostics refuses every option, since the mapper takes none.
func optionDiagnostics(options map[string]json.RawMessage) []OptionDiagnostic {
	diags := []OptionDiagnostic{}
	for _, key := range slices.Sorted(maps.Keys(options)) {
		diags = append(diags, OptionDiagnostic{
			Path:        []string{key},
			MessageText: fmt.Sprintf("the tdl content mapper takes no options; remove %q", key),
			Code:        CodeOption,
		})
	}
	return diags
}

type transformParams struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// Module is what a transform returns: a .tdl file as a module of types.
type Module struct {
	Extension   string       `json:"extension"`
	Text        string       `json:"text"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic is a problem in the .tdl file, at a range of its UTF-8 bytes.
type Diagnostic struct {
	MessageText string `json:"messageText"`
	Code        int    `json:"code"`
	Start       int    `json:"start"`
	Length      int    `json:"length"`
}

// Transform returns the module for the .tdl file named name holding
// content.
func (m Mapper) Transform(name, content string) Module {
	t := transform{name: name, lines: newLines(content), mod: Module{Extension: ".mts", Text: emptyModule, Diagnostics: []Diagnostic{}}}

	file, err := parser.Parse(name, strings.NewReader(content))
	if err != nil {
		list, ok := errors.AsType[parser.ErrorList](err)
		if !ok {
			t.at(0, 1, CodeSyntax, err.Error())
		}
		for _, e := range list {
			t.at(e.Pos.Offset, 1, CodeSyntax, e.Msg)
		}
		return t.mod
	}

	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.lowering(diags)
		return t.mod
	}

	desc := Backend{}.Describe()
	if problems := host.CheckDirectives(Name, model, desc); t.report(problems, CodeDirective) {
		return t.mod
	}
	req, problems := analysis.Request(model, desc, Name, "")
	if t.report(problems, CodeModel) {
		return t.mod
	}

	resp := generate(req, true)
	if files := resp.GetFiles(); !t.report(resp.GetDiagnostics(), CodeSkipped) && len(files) > 0 {
		t.mod.Text = string(files[0].GetContent())
	}
	return t.mod
}

// transform is one file's module as it is built.
type transform struct {
	name  string
	lines lines
	mod   Module
}

func (t *transform) at(start, length, code int, msg string) {
	t.mod.Diagnostics = append(t.mod.Diagnostics, Diagnostic{MessageText: msg, Code: code, Start: start, Length: max(length, 1)})
}

// lowering reports what lowering found at its range. Lowering reports a
// file it imports that does not parse or lower at the import.
func (t *transform) lowering(diags sema.Diagnostics) {
	for _, d := range diags {
		start, length := 0, 1
		if d.Pos.Filename == t.name {
			start, length = d.Pos.Offset, max(d.End.Offset-d.Pos.Offset, 1)
		}
		t.at(start, length, CodeLowering, d.Msg)
	}
}

// report adds each diagnostic tsc should see under code and returns
// whether any was an error, which leaves the module empty. A warning with
// a loss code is dropped: tsc reports every mapper diagnostic as an error,
// and the losses are tdl gen's to report.
func (t *transform) report(diags []*plugin.Diagnostic, code int) bool {
	for _, d := range diags {
		if d.GetSeverity() == plugin.Severity_SEVERITY_WARNING && d.GetCode() != "" {
			continue
		}
		start, length := t.lines.span(t.name, d.GetPosition())
		t.at(start, length, code, d.GetMessage())
	}
	fatal := host.Fatal(diags)
	if fatal {
		t.mod.Text = emptyModule
	}
	return fatal
}

// lines is the byte offset at which each line of a file starts.
type lines []int

func newLines(content string) lines {
	ls := lines{0}
	for i := range len(content) {
		if content[i] == '\n' {
			ls = append(ls, i+1)
		}
	}
	return ls
}

// offset is the byte offset of a 1-based line and column, or -1 for one
// outside the file.
func (ls lines) offset(line, col int32) int {
	if line < 1 || int(line) > len(ls) || col < 1 {
		return -1
	}
	return ls[line-1] + int(col) - 1
}

// span is the range pos covers in the file named name. A position in
// another file, or none, is the file's start.
func (ls lines) span(name string, pos *ir.Position) (start, length int) {
	if pos == nil || (pos.GetFilename() != "" && pos.GetFilename() != name) {
		return 0, 1
	}
	start = ls.offset(pos.GetLine(), pos.GetColumn())
	if start < 0 {
		return 0, 1
	}
	if end := ls.offset(pos.GetEndLine(), pos.GetEndColumn()); end > start {
		return start, end - start
	}
	return start, 1
}
