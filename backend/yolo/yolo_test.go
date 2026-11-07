package yolo_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/yolo"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// record re-records every case against Anthropic's API, which needs
// ANTHROPIC_API_KEY, and rewrites what each case wants.
var record = flag.Bool("record", false, "re-record testdata/yolo against the live API")

const (
	cases = "../../testdata/yolo"
	smoke = "../../testdata/gen/smoke/source.tdl"

	// wantDir holds the files a case generates, and golden its
	// diagnostics.
	wantDir = "want"
	golden  = "diagnostics.golden"
)

// Source is a case's model: its source.tdl, or for a case holding only a
// target.tdl, the smoke model with that target block appended, which is
// what checks.gen-yolo generates from.
func Source(t *testing.T, dir string) string {
	t.Helper()
	if data, err := os.ReadFile(filepath.Join(dir, "source.tdl")); err == nil {
		return string(data)
	}
	base, err := os.ReadFile(smoke)
	if err != nil {
		t.Fatal(err)
	}
	block, err := os.ReadFile(filepath.Join(dir, "target.tdl"))
	if err != nil {
		t.Fatal(err)
	}
	return string(base) + "\n" + string(block)
}

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("source.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	return model
}

func request(t *testing.T, src string) *plugin.Request {
	t.Helper()
	req, diags := analysis.Request(lower(t, src), yolo.Backend{}.Describe(), yolo.Name, "out")
	if len(diags) > 0 {
		t.Fatalf("request: %v", diags)
	}
	return req
}

// Each case in testdata/yolo replays its conversation and generates what
// it wants, with the diagnostics it wants.
func TestCases(t *testing.T) {
	dirs, err := os.ReadDir(cases)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join(cases, d.Name())
		t.Run(d.Name(), func(t *testing.T) {
			req := request(t, Source(t, dir))

			var provider yolo.Provider
			var recorder *yolo.Recorder
			if *record {
				key := os.Getenv(yolo.KeyEnv)
				if key == "" {
					t.Fatalf("-record needs %s", yolo.KeyEnv)
				}
				recorder = &yolo.Recorder{Provider: yolo.NewAnthropic(key)}
				provider = recorder
			} else {
				r, err := yolo.LoadReplay(dir)
				if err != nil {
					t.Fatal(err)
				}
				provider = r
			}

			resp, err := yolo.Backend{Provider: provider}.Generate(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if *record {
				if err := recorder.Save(dir); err != nil {
					t.Fatal(err)
				}
				writeWant(t, dir, resp)
				return
			}
			compareWant(t, dir, resp)
		})
	}
}

func diagnostics(resp *plugin.Response) string {
	var b strings.Builder
	for _, d := range resp.GetDiagnostics() {
		sev := strings.ToLower(strings.TrimPrefix(d.GetSeverity().String(), "SEVERITY_"))
		fmt.Fprintf(&b, "%d:%d %s", d.GetPosition().GetLine(), d.GetPosition().GetColumn(), sev)
		if d.GetCode() != "" {
			fmt.Fprintf(&b, " %s", d.GetCode())
		}
		fmt.Fprintf(&b, ": %s\n", d.GetMessage())
	}
	return b.String()
}

func writeWant(t *testing.T, dir string, resp *plugin.Response) {
	t.Helper()
	want := filepath.Join(dir, wantDir)
	if err := os.RemoveAll(want); err != nil {
		t.Fatal(err)
	}
	for _, f := range resp.GetFiles() {
		path := filepath.Join(want, f.GetPath())
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, golden), []byte(diagnostics(resp)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func compareWant(t *testing.T, dir string, resp *plugin.Response) {
	t.Helper()
	wantDiags, err := os.ReadFile(filepath.Join(dir, golden))
	if err != nil {
		t.Fatal(err)
	}
	if got := diagnostics(resp); got != string(wantDiags) {
		t.Errorf("diagnostics:\n%s\nwant:\n%s", got, wantDiags)
	}

	want := map[string]string{}
	root := filepath.Join(dir, wantDir)
	_ = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		want[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	got := map[string]string{}
	for _, f := range resp.GetFiles() {
		got[f.GetPath()] = string(f.GetContent())
	}
	for path, content := range want {
		if got[path] != content {
			t.Errorf("%s:\n%s\nwant:\n%s", path, got[path], content)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			t.Errorf("generated %s, which the case does not want", path)
		}
	}
}

// fixed is a provider answering every turn with the next reply it holds,
// sending nothing anywhere.
type fixed struct {
	replies []yolo.Message
	seen    []*yolo.Conversation
}

func (f *fixed) Send(_ context.Context, c *yolo.Conversation) (yolo.Message, error) {
	f.seen = append(f.seen, c)
	if len(f.replies) == 0 {
		return yolo.Message{}, fmt.Errorf("no reply left")
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	return r, nil
}

const order = `package shop

type Order: Entity {
  id: string
}

target yolo for shop {
  language("elixir")
}
`

func call(name, input string) yolo.Call {
	return yolo.Call{ID: name, Name: name, Input: []byte(input)}
}

// The host keeps a nondeterministic backend's output while the request is
// unchanged, so regenerating never calls the model by accident.
func TestNondeterministic(t *testing.T) {
	if !(yolo.Backend{}).Describe().Nondeterministic {
		t.Error("yolo does not declare itself nondeterministic")
	}
}

// Without a key or a recording, nothing is sent and nothing written.
func TestMissingKeyWritesNothing(t *testing.T) {
	t.Setenv(yolo.ReplayEnv, "")
	t.Setenv(yolo.KeyEnv, "")
	resp, err := yolo.Backend{}.Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFiles()) != 0 {
		t.Errorf("wrote %d files", len(resp.GetFiles()))
	}
	if got := diagnostics(resp); !strings.Contains(got, "error") || !strings.Contains(got, yolo.KeyEnv) {
		t.Errorf("diagnostics: %s", got)
	}
}

func TestLanguageIsRequired(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`, `prompt("anything")`, 1)
	f := &fixed{}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) != 0 || len(resp.GetFiles()) != 0 {
		t.Errorf("sent %d turns and wrote %d files", len(f.seen), len(resp.GetFiles()))
	}
	if got := diagnostics(resp); !strings.Contains(got, "language(...)") {
		t.Errorf("diagnostics: %s", got)
	}
}

// Every path that leaves the output directory is an error that writes
// nothing, even after a safe file.
func TestUnsafePathsWriteNothing(t *testing.T) {
	for _, path := range []string{"/etc/passwd", "../out.ex", "a/../../b.ex", "a/../b.ex", `a\b.ex`, ""} {
		t.Run(path, func(t *testing.T) {
			f := &fixed{replies: []yolo.Message{{
				Role: yolo.Assistant,
				Calls: []yolo.Call{
					call("write_file", `{"path": "ok.ex", "content": "x"}`),
					call("write_file", fmt.Sprintf(`{"path": %q, "content": "x"}`, path)),
				},
				Stop: yolo.StopTool,
			}}}
			resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.GetFiles()) != 0 {
				t.Errorf("wrote %d files", len(resp.GetFiles()))
			}
			if got := diagnostics(resp); !strings.Contains(got, "error") {
				t.Errorf("diagnostics: %s", got)
			}
		})
	}
}

func TestSafePath(t *testing.T) {
	for path, want := range map[string]bool{
		"shop.ex":         true,
		"lib/shop/o.ex":   true,
		"./shop.ex":       true,
		"":                false,
		"/shop.ex":        false,
		"../shop.ex":      false,
		"lib/../shop.ex":  false,
		`lib\shop.ex`:     false,
		"lib/..":          false,
		"lib/..something": true,
	} {
		if got := yolo.SafePath(path); got != want {
			t.Errorf("SafePath(%q) = %v, want %v", path, got, want)
		}
	}
}

// A refused call goes back to the model as an error result, and the
// conversation goes on.
func TestRefusedCallsGoBackToTheModel(t *testing.T) {
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("report_loss", `{"decl": "Order", "code": "lossy.nonesuch", "message": "m"}`),
			call("report_loss", `{"decl": "Nonesuch", "code": "lossy.key", "message": "m"}`),
			call("write_file", `{"path": 1}`),
			call("delete_file", `{}`),
		}},
		{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("write_file", `{"path": "shop.ex", "content": "defmodule Shop do\nend\n"}`),
			call("report_loss", `{"decl": "Order.id", "code": "lossy.key", "message": "No key."}`),
		}},
		{Role: yolo.Assistant, Stop: yolo.StopEnd, Text: "Done."},
	}}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) != 3 {
		t.Fatalf("sent %d turns, want 3", len(f.seen))
	}
	results := f.seen[1].Messages[2].Results
	if len(results) != 4 {
		t.Fatalf("%d results, want 4", len(results))
	}
	for i, r := range results {
		if !r.Error {
			t.Errorf("result %d was not refused: %s", i, r.Content)
		}
	}
	if !strings.Contains(results[0].Content, emit.LossKey) {
		t.Errorf("an unknown code's refusal does not list the codes: %s", results[0].Content)
	}
	if got, want := diagnostics(resp), "4:3 warning lossy.key: No key.\n"; got != want {
		t.Errorf("diagnostics: %q, want %q", got, want)
	}
	if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "shop.ex" {
		t.Errorf("files: %v", resp.GetFiles())
	}
}

func TestStopsAreErrors(t *testing.T) {
	for _, stop := range []yolo.Stop{yolo.StopRefused, yolo.StopLength} {
		t.Run(string(stop), func(t *testing.T) {
			f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: stop, Calls: []yolo.Call{
				call("write_file", `{"path": "shop.ex", "content": "x"}`),
			}}}}
			resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.GetFiles()) != 0 || !strings.Contains(diagnostics(resp), "error") {
				t.Errorf("files %v, diagnostics %s", resp.GetFiles(), diagnostics(resp))
			}
		})
	}
}

func TestNoFilesIsAnError(t *testing.T) {
	f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: yolo.StopEnd, Text: "I can't."}}}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diagnostics(resp), "wrote no files") {
		t.Errorf("diagnostics: %s", diagnostics(resp))
	}
}

// Two spellings of one path are one file, the last write winning.
func TestPathsAreCleaned(t *testing.T) {
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("write_file", `{"path": "./lib//shop.ex", "content": "first"}`),
			call("write_file", `{"path": "lib/shop.ex", "content": "second"}`),
		}},
		{Role: yolo.Assistant, Stop: yolo.StopEnd},
	}}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	files := resp.GetFiles()
	if len(files) != 1 || files[0].GetPath() != "lib/shop.ex" || string(files[0].GetContent()) != "second" {
		t.Errorf("files: %v", files)
	}
}

// A model that never stops calling tools is cut off.
func TestTurnsAreBounded(t *testing.T) {
	f := &fixed{}
	for range 100 {
		f.replies = append(f.replies, yolo.Message{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("write_file", `{"path": "shop.ex", "content": "x"}`),
		}})
	}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) == 100 || !strings.Contains(diagnostics(resp), "still calling tools") {
		t.Errorf("sent %d turns; diagnostics %s", len(f.seen), diagnostics(resp))
	}
}

// The prompt carries the language, the instructions, the model as TDL,
// the directives chosen, every loss code, and the spec.
func TestPrompt(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`,
		`language("elixir")
  prompt("Ecto embedded schemas")
  model("claude-sonnet-5-5")`, 1)
	f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
		call("write_file", `{"path": "shop.ex", "content": "x"}`),
	}}, {Role: yolo.Assistant, Stop: yolo.StopEnd}}}
	if _, err := (yolo.Backend{Provider: f}).Generate(context.Background(), request(t, src)); err != nil {
		t.Fatal(err)
	}
	c := f.seen[0]
	if c.Model != "claude-sonnet-5-5" {
		t.Errorf("model %q", c.Model)
	}
	first := c.Messages[0].Text
	for _, want := range []string{"Write this model in elixir.", "Ecto embedded schemas", "type Order: Entity {", "target yolo for shop {"} {
		if !strings.Contains(first, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, first)
		}
	}
	for _, code := range emit.LossCodes {
		if !strings.Contains(c.System, "- "+code+": ") {
			t.Errorf("the system prompt does not explain %s", code)
		}
	}
	if !strings.Contains(c.System, "# TDL") && !strings.Contains(c.System, "TDL specification") {
		t.Error("the system prompt lacks the spec")
	}
	var names []string
	for _, tool := range c.Tools {
		names = append(names, tool.Name)
	}
	if !slices.Equal(names, []string{"write_file", "report_loss"}) {
		t.Errorf("tools %v", names)
	}
}

func TestDefaultModel(t *testing.T) {
	f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: yolo.StopEnd}}}
	_, _ = yolo.Backend{Provider: f}.Generate(context.Background(), request(t, order))
	if f.seen[0].Model != yolo.DefaultModel {
		t.Errorf("model %q", f.seen[0].Model)
	}
}

// A replay refuses a conversation other than the one it recorded.
func TestReplayRefusesAnotherConversation(t *testing.T) {
	r, err := yolo.LoadReplay(filepath.Join(cases, "order"))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Replace(order, `"elixir"`, `"kotlin"`, 1)
	resp, err := yolo.Backend{Provider: r}.Generate(context.Background(), request(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if got := diagnostics(resp); !strings.Contains(got, "re-record") {
		t.Errorf("diagnostics: %s", got)
	}
}
