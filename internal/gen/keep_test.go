package gen_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// fickle answers every request with different content, as a backend that
// hands the mapping to a language model does.
type fickle struct {
	calls int
	warn  string // a warning to report with every response, if set
}

func (f *fickle) Describe() plugin.Description {
	return plugin.Description{Name: "fickle", Nondeterministic: true}
}

func (f *fickle) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	f.calls++
	resp := &plugin.Response{Files: []*plugin.File{{
		Path:    "out.txt",
		Content: fmt.Appendf(nil, "call %d over %d declarations\n", f.calls, len(req.GetModel().GetDecls())),
	}}}
	if f.warn != "" {
		resp.Diagnostics = []*plugin.Diagnostic{{
			Severity: plugin.Severity_SEVERITY_WARNING,
			Message:  f.warn,
			Code:     "lossy.constraint",
		}}
	}
	return resp, nil
}

func lowerSource(t *testing.T, name, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse(name, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags)
	}
	return model
}

const keepSource = `package shop

type Money {
  cents: int64
}
`

func runFickle(t *testing.T, b *fickle, out, src string, mode gen.Mode) gen.Result {
	t.Helper()
	target := gen.Target{Name: "fickle", Out: out}
	result, err := gen.Run(context.Background(), b, target, lowerSource(t, "shop.tdl", src), mode)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return result
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The same request keeps the files a nondeterministic backend wrote, and
// does not ask it again; --verify passes without asking it at all.
func TestUnchangedRequestKeepsOutput(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	first := runFickle(t, b, out, keepSource, gen.ModeWrite)
	if len(first.Written) != 1 || b.calls != 1 {
		t.Fatalf("first run wrote %v after %d calls", first.Written, b.calls)
	}
	want := read(t, filepath.Join(out, "out.txt"))

	again := runFickle(t, b, out, keepSource, gen.ModeWrite)
	if b.calls != 1 || len(again.Written) != 0 || len(again.Kept) != 1 {
		t.Errorf("second run: %d calls, wrote %v, kept %v", b.calls, again.Written, again.Kept)
	}
	if got := read(t, filepath.Join(out, "out.txt")); got != want {
		t.Errorf("kept file changed: %q, want %q", got, want)
	}

	verify := runFickle(t, b, out, keepSource, gen.ModeVerify)
	if b.calls != 1 || len(verify.Stale) != 0 || len(verify.Expected) != 1 {
		t.Errorf("verify: %d calls, stale %v, expected %v", b.calls, verify.Stale, verify.Expected)
	}
}

// Where the model is written, and so every position in the request, does
// not change the hash: moving a declaration down a line keeps the output.
func TestPositionsDoNotChangeTheRequest(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	runFickle(t, b, out, "// a comment\n\n"+keepSource, gen.ModeWrite)
	if b.calls != 1 {
		t.Errorf("moving the model asked the backend again: %d calls", b.calls)
	}
}

// A changed request is stale under --verify without asking the backend, and
// a write regenerates and records the new request.
func TestChangedRequestRegenerates(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	changed := keepSource + "\ntype Sku {\n  code: string\n}\n"

	verify := runFickle(t, b, out, changed, gen.ModeVerify)
	if b.calls != 1 || len(verify.Stale) != 1 || !strings.Contains(verify.Stale[0].Reason, "request changed") {
		t.Errorf("verify: %d calls, stale %v", b.calls, verify.Stale)
	}
	if len(verify.Expected) != 1 {
		t.Errorf("verify expected %v, want the kept file so it is not an orphan", verify.Expected)
	}

	written := runFickle(t, b, out, changed, gen.ModeWrite)
	if b.calls != 2 || len(written.Written) != 1 {
		t.Errorf("write: %d calls, wrote %v", b.calls, written.Written)
	}
	if verify := runFickle(t, b, out, changed, gen.ModeVerify); len(verify.Stale) != 0 {
		t.Errorf("verify after regenerating: %v", verify.Stale)
	}
}

// A kept file edited by hand fails --verify, and a write replaces it.
func TestEditedKeptFile(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	path := filepath.Join(out, "out.txt")
	if err := os.WriteFile(path, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	verify := runFickle(t, b, out, keepSource, gen.ModeVerify)
	if b.calls != 1 || len(verify.Stale) != 1 || verify.Stale[0].Reason != "differs" {
		t.Errorf("verify: %d calls, stale %v", b.calls, verify.Stale)
	}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	if b.calls != 2 || read(t, path) == "edited\n" {
		t.Errorf("write: %d calls, file %q", b.calls, read(t, path))
	}
}

// A target never generated is stale under --verify, without asking.
func TestNeverGeneratedIsStale(t *testing.T) {
	b := &fickle{}
	verify := runFickle(t, b, t.TempDir(), keepSource, gen.ModeVerify)
	if b.calls != 0 || len(verify.Stale) != 1 {
		t.Errorf("verify: %d calls, stale %v", b.calls, verify.Stale)
	}
}

// --clean always asks the backend again.
func TestCleanRegenerates(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	runFickle(t, b, out, keepSource, gen.ModeClean)
	if b.calls != 2 {
		t.Errorf("clean made %d calls, want 2", b.calls)
	}
	if verify := runFickle(t, b, out, keepSource, gen.ModeVerify); len(verify.Stale) != 0 {
		t.Errorf("verify after clean: %v", verify.Stale)
	}
}

// The record sits in the marker as comments, so the files stay listed for
// --clean, and a deterministic backend writing there leaves it alone.
func TestKeptRecordSurvivesOtherWrites(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	write(t, out, &plugin.File{Path: "other.txt", Content: []byte("x")})

	if again := runFickle(t, b, out, keepSource, gen.ModeWrite); b.calls != 1 || len(again.Kept) != 1 {
		t.Errorf("after another write: %d calls, kept %v", b.calls, again.Kept)
	}
	owned, err := gen.Owned(out)
	if err != nil || !owned[filepath.Join(out, "out.txt")] || !owned[filepath.Join(out, "other.txt")] {
		t.Errorf("owned = %v, %v", owned, err)
	}
	removed, err := gen.Clean(out)
	if err != nil || len(removed) != 2 {
		t.Errorf("clean removed %v, %v", removed, err)
	}
	if _, ok, _ := gen.Recorded(out, "fickle"); ok {
		t.Error("clean left the record behind")
	}
}

// Kept files are reported with the warnings the backend gave when it wrote
// them, so a lossy generation stays visible on every later run.
func TestKeptOutputKeepsItsWarnings(t *testing.T) {
	out, b := t.TempDir(), &fickle{warn: "dropped a constraint"}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	b.warn = ""

	for _, mode := range []gen.Mode{gen.ModeWrite, gen.ModeVerify} {
		result := runFickle(t, b, out, keepSource, mode)
		if b.calls != 1 || len(result.Diagnostics) != 1 || result.Diagnostics[0].GetMessage() != "dropped a constraint" || result.Diagnostics[0].GetCode() != "lossy.constraint" {
			t.Errorf("mode %v: %d calls, diagnostics %v", mode, b.calls, result.Diagnostics)
		}
	}
}

// A record that lost a file line no longer says what was written, so it is
// stale under --verify and a write regenerates.
func TestIncompleteRecordRegenerates(t *testing.T) {
	out, b := t.TempDir(), &fickle{}
	runFickle(t, b, out, keepSource, gen.ModeWrite)
	marker := filepath.Join(out, gen.MarkerName)
	var kept []string
	for line := range strings.Lines(read(t, marker)) {
		if !strings.HasPrefix(line, "#@ fickle file ") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(marker, []byte(strings.Join(kept, "")), 0o644); err != nil {
		t.Fatal(err)
	}

	verify := runFickle(t, b, out, keepSource, gen.ModeVerify)
	if b.calls != 1 || len(verify.Stale) != 1 || !strings.Contains(verify.Stale[0].Reason, "incomplete") {
		t.Errorf("verify: %d calls, stale %v", b.calls, verify.Stale)
	}
	if runFickle(t, b, out, keepSource, gen.ModeWrite); b.calls != 2 {
		t.Errorf("write made %d calls, want 2", b.calls)
	}
}
