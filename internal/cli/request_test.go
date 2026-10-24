package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// recorder keeps every request gen sends it and writes nothing.
type recorder struct {
	plugin.Backend
	requests *[]*plugin.Request
}

func (r recorder) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	*r.requests = append(*r.requests, req)
	return &plugin.Response{}, nil
}

// described is a backend tdl has neither compiled in nor found on PATH.
type described struct{ plugin.Description }

func (d described) Describe() plugin.Description { return d.Description }

func (described) Generate(context.Context, *plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{}, nil
}

// testResolver serves the compiled-in backends, a plugin on PATH, or a
// backend declaring nothing, so every target name resolves.
func testResolver(name string) (plugin.Backend, error) {
	if b, err := gen.Resolve(name); err == nil {
		return b, nil
	}
	return described{plugin.Description{Name: name}}, nil
}

// readRequestSet decodes what request wrote, checking the handshake.
func readRequestSet(t *testing.T, out string) *plugin.RequestSet {
	t.Helper()
	conn := plugin.NewConn(strings.NewReader(out), nil)
	var hello plugin.Handshake
	if err := conn.Recv(&hello); err != nil {
		t.Fatalf("reading the handshake: %v", err)
	}
	if hello.GetFramingVersion() != plugin.FramingVersion || hello.GetIrVersion() != plugin.IRVersion {
		t.Errorf("handshake = %v", &hello)
	}
	var set plugin.RequestSet
	if err := conn.Recv(&set); err != nil {
		t.Fatalf("reading the request set: %v", err)
	}
	return &set
}

// runRequest runs request with stdin and decodes what it wrote.
func runRequest(t *testing.T, stdin []byte, args ...string) *plugin.RequestSet {
	t.Helper()
	cmd := requestCmd(testResolver)
	cmd.SetIn(bytes.NewReader(stdin))
	out, errOut, err := run(t, cmd, args...)
	if err != nil {
		t.Fatalf("request: %v\n%s", err, errOut)
	}
	return readRequestSet(t, out)
}

// overlay frames an Overlay holding path and text pairs.
func overlay(t *testing.T, pairs ...string) []byte {
	t.Helper()
	var o plugin.Overlay
	for i := 0; i < len(pairs); i += 2 {
		o.Files = append(o.Files, &plugin.OverlayFile{Path: pairs[i], Text: pairs[i+1]})
	}
	var buf bytes.Buffer
	if err := plugin.NewConn(nil, &buf).Send(&o); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// request sends each target in the smoke model what gen sends it.
func TestRequestMatchesGen(t *testing.T) {
	src, err := os.ReadFile("../../testdata/gen/smoke/source.tdl")
	if err != nil {
		t.Fatal(err)
	}
	// gen writes beside the source, so it runs over a copy.
	path := filepath.Join(t.TempDir(), "source.tdl")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}

	targets := []string{"csharp", "protobuf", "typescript", "haskell"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var sent []*plugin.Request
			resolve := func(name string) (plugin.Backend, error) {
				b, err := testResolver(name)
				return recorder{b, &sent}, err
			}
			if _, errOut, err := run(t, genCmd(resolve), "--target", target, path); err != nil {
				t.Fatalf("gen: %v\n%s", err, errOut)
			}

			set := runRequest(t, nil, "--target", target, path)
			if len(sent) != 1 || len(set.GetRequests()) != 1 {
				t.Fatalf("gen sent %d requests and request wrote %d, want 1 each", len(sent), len(set.GetRequests()))
			}
			if !proto.Equal(sent[0], set.GetRequests()[0]) {
				t.Error("request wrote a different request from the one gen sent")
			}
		})
	}
}

// An overlay text is read in place of the file on disk, both for a file
// requested and for one it imports, and a file only in the overlay is
// requested too.
func TestRequestReadsTheOverlay(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tdl")
	common := filepath.Join(dir, "common.tdl")
	write := func(path, src string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(main, "package p\n\nimport \"common.tdl\" as common\n\ntype OnDisk {\n  a: common.Shared\n}\n\ntarget csharp for p {}\n")
	write(common, "package q\n\ntype Shared {\n  id: string\n}\n")

	set := runRequest(t,
		overlay(t,
			main, "package p\n\nimport \"common.tdl\" as common\n\ntype Edited {\n  a: common.Unsaved\n}\n\ntarget csharp for p {}\n",
			common, "package q\n\ntype Unsaved {\n  id: string\n}\n",
			filepath.Join(dir, "new.tdl"), "package n\n\ntype Fresh {\n  id: string\n}\n\ntarget csharp for n {}\n",
		),
		"--target", "csharp", "--overlay", main)

	if len(set.GetDiagnostics()) > 0 {
		t.Fatalf("diagnostics: %v", set.GetDiagnostics())
	}
	var names []string
	for _, req := range set.GetRequests() {
		for _, d := range req.GetModel().GetDecls() {
			if strings.HasPrefix(d.GetMeta().GetPosition().GetFilename(), dir) {
				names = append(names, d.GetMeta().GetName())
			}
		}
	}
	got := strings.Join(names, " ")
	for _, want := range []string{"Edited", "Fresh"} {
		if !strings.Contains(got, want) {
			t.Errorf("declarations %q lack %s", got, want)
		}
	}
	if strings.Contains(got, "OnDisk") {
		t.Errorf("declarations %q were read from disk", got)
	}
}

// A file that does not lower yields no request and an error at its range,
// and the other files still yield theirs.
func TestRequestReportsErrors(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.tdl")
	broken := filepath.Join(dir, "broken.tdl")
	unparsed := filepath.Join(dir, "unparsed.tdl")
	for path, src := range map[string]string{
		good:     "package p\n\ntype A {\n  id: string\n}\n\ntarget csharp for p {}\n",
		broken:   "package p\n\ntype B {\n  id: Missing\n}\n\ntarget csharp for p {}\n",
		unparsed: "package p\n\ntype {\n",
	} {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	set := runRequest(t, nil, "--target", "csharp", good, broken, unparsed, filepath.Join(dir, "absent.tdl"))

	if len(set.GetRequests()) != 1 || set.GetRequests()[0].GetModel().GetPackage() != "p" {
		t.Fatalf("got %d requests, want good.tdl's alone", len(set.GetRequests()))
	}
	files := map[string]*ir.Position{}
	for _, d := range set.GetDiagnostics() {
		if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR {
			t.Errorf("not an error: %v", d)
		}
		files[filepath.Base(d.GetPosition().GetFilename())] = d.GetPosition()
	}
	want := &ir.Position{Filename: broken, Line: 4, Column: 7, EndLine: 4, EndColumn: 14}
	if !proto.Equal(files["broken.tdl"], want) {
		t.Errorf("broken.tdl's error is at %v, want %v", files["broken.tdl"], want)
	}
	for _, name := range []string{"unparsed.tdl", "absent.tdl"} {
		if files[name] == nil {
			t.Errorf("no error for %s", name)
		}
	}
}

// A warning is reported beside the request, which it does not stop.
func TestRequestKeepsWarnings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.tdl")
	src := "package p\n\ntype A {\n  id: string\n}\n\ntarget csharp for p {\n  unknown(\"x\")\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	set := runRequest(t, nil, "--target", "csharp", path)
	if len(set.GetRequests()) != 1 || len(set.GetDiagnostics()) != 1 {
		t.Fatalf("got %d requests and %v, want one request and the undeclared directive's warning", len(set.GetRequests()), set.GetDiagnostics())
	}
}
