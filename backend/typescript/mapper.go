package typescript

// The content mapper lets TypeScript 7.1 import a .tdl file as a module of
// types: tsc spawns `tdl-gen-typescript content-mapper` and asks it, over
// JSON-RPC on standard input and output, for each .tdl file's module.
// docs/design/typescript-content-mapper.md describes the protocol.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
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

// Serve answers requests read from r on w until r ends.
func (m Mapper) Serve(r io.Reader, w io.Writer) error {
	in := bufio.NewReader(r)
	for {
		body, err := readFrame(in)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return fmt.Errorf("content mapper: %w", err)
		}
		resp := m.answer(req)
		if req.ID == nil {
			continue // a notification
		}
		if err := writeFrame(w, resp); err != nil {
			return err
		}
	}
}

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (m Mapper) answer(req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	result, err := m.handle(req.Method, req.Params)
	if err != nil {
		resp.Error = &rpcError{Code: -32603, Message: err.Error()}
		return resp
	}
	resp.Result = result
	return resp
}

func (m Mapper) handle(method string, params json.RawMessage) (any, error) {
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
	return nil, fmt.Errorf("unknown method %s", method)
}

func unmarshal(params json.RawMessage, v any) error {
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

// readFrame reads one message framed by a Content-Length header.
func readFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && line == "" && length < 0 {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("content mapper: reading a header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, fmt.Errorf("content mapper: bad Content-Length %q", value)
			}
			if length >= 0 && n != length {
				return nil, fmt.Errorf("content mapper: conflicting Content-Length headers %d and %d", length, n)
			}
			length = n
		}
	}
	if length < 0 {
		return nil, errors.New("content mapper: a message has no Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("content mapper: reading a message: %w", err)
	}
	return body, nil
}

// writeFrame writes one message with its Content-Length header.
func writeFrame(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}
