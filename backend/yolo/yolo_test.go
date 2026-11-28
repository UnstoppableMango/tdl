package yolo_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/yolo"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// record re-records every case against its provider's live API, which
// needs ANTHROPIC_API_KEY, or OPENAI_BASE_URL or OPENAI_API_KEY for a
// provider("openai") case, and rewrites what each case wants.
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
	return lowerAt(t, "source.tdl", src)
}

// lowerAt lowers src as the file at path, so the backend finds the
// tdl.toml beside it.
func lowerAt(t *testing.T, path, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	return model
}

// caseRequest is a case's request as tdl gen sends it: a source.tdl is
// lowered from disk, so its imports resolve, with what each dependency
// declares and generated.
func caseRequest(t *testing.T, dir string) *plugin.Request {
	t.Helper()
	path := filepath.Join(dir, "source.tdl")
	data, err := os.ReadFile(path)
	if err != nil {
		return request(t, Source(t, dir))
	}
	file, err := parser.Parse(path, strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	out := filepath.Join(dir, "out")
	if targets, err := gen.Targets(model, ""); err == nil {
		out = targets[0].Out
	}
	req, problems := analysis.Request(model, yolo.Backend{}.Describe(), yolo.Name, out)
	if len(problems) > 0 {
		t.Fatalf("request: %v", problems)
	}
	if _, err := gen.Depend(req, sema.FSLoader{}); err != nil {
		t.Fatal(err)
	}
	return req
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
			src := Source(t, dir)
			req := caseRequest(t, dir)

			var provider yolo.Provider
			var recorder *yolo.Recorder
			if *record {
				name := yolo.ProviderAnthropic
				if m := providerDirective.FindStringSubmatch(src); m != nil {
					name = m[1]
				}
				live, err := yolo.Live(name)
				if err != nil {
					t.Fatalf("-record: %v", err)
				}
				recorder = &yolo.Recorder{Provider: live}
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

// providerDirective finds the provider a case's target block names.
var providerDirective = regexp.MustCompile(`provider\("(\w+)"\)`)

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
	err = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if path == root && errors.Is(err, fs.ErrNotExist) {
			return fs.SkipAll
		}
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
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range resp.GetFiles() {
		got[f.GetPath()] = string(f.GetContent())
	}
	for path, content := range want {
		if actual, ok := got[path]; !ok || actual != content {
			t.Errorf("%s:\n%s\nwant:\n%s", path, actual, content)
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

// The openai provider needs an endpoint or a key, a model, and an effort
// it can send; without one, nothing is sent and nothing written.
func TestOpenAIRefusesBeforeSending(t *testing.T) {
	openai := func(directives string) string {
		return strings.Replace(order, `language("elixir")`, `language("elixir")
  provider("openai")
  `+directives, 1)
	}
	for name, tc := range map[string]struct {
		src, env, want string
	}{
		"no endpoint or key": {openai(`model("gpt-5.5")`), "", yolo.OpenAIBaseURLEnv},
		"no model":           {openai(`effort("low")`), "set", `model(...)`},
		"xhigh":              {openai(`model("gpt-5.5") effort("xhigh")`), "set", "low, medium, high"},
		"max":                {openai(`model("gpt-5.5") effort("max")`), "set", `not "max"`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(yolo.ReplayEnv, "")
			t.Setenv(yolo.OpenAIBaseURLEnv, tc.env)
			t.Setenv(yolo.OpenAIKeyEnv, "")
			resp, err := yolo.Backend{}.Generate(context.Background(), request(t, tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.GetFiles()) != 0 {
				t.Errorf("wrote %d files", len(resp.GetFiles()))
			}
			if got := diagnostics(resp); !strings.Contains(got, "error") || !strings.Contains(got, tc.want) {
				t.Errorf("diagnostics: %s", got)
			}
			if name == "no endpoint or key" && !strings.Contains(diagnostics(resp), yolo.OpenAIKeyEnv) {
				t.Errorf("the error does not name %s", yolo.OpenAIKeyEnv)
			}
		})
	}
}

// A provider the host did not check is still refused before anything is
// sent.
func TestUnknownProviderIsRefused(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`, `language("elixir") provider("gemini")`, 1)
	f := &fixed{}
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), request(t, src))
	if err == nil && len(f.seen) == 0 && strings.Contains(diagnostics(resp), `"gemini"`) {
		return
	}
	t.Errorf("sent %d turns, %v: %s", len(f.seen), err, diagnostics(resp))
}

// A base URL alone is enough, since a local server needs no key.
func TestOpenAIBaseURLNeedsNoKey(t *testing.T) {
	t.Setenv(yolo.OpenAIBaseURLEnv, "http://localhost:11434/v1")
	t.Setenv(yolo.OpenAIKeyEnv, "")
	if _, err := yolo.Live(yolo.ProviderOpenAI); err != nil {
		t.Error(err)
	}
}

// Under openai, the model is the one written and no effort is sent unless
// one is; Anthropic keeps its defaults.
func TestOpenAIModelAndEffort(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`, `language("elixir") provider("openai") model("qwen3")`, 1)
	f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: yolo.StopEnd}}}
	_, _ = yolo.Backend{Provider: f}.Generate(context.Background(), request(t, src))
	if f.seen[0].Model != "qwen3" || f.seen[0].Effort != "" {
		t.Errorf("model %q, effort %q", f.seen[0].Model, f.seen[0].Effort)
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
		".":               true,
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
			call("write_file", `{"path": "./", "content": "x"}`),
			call("report_loss", `{"decl": "Order", "code": "lossy.key"}`),
			call("report_loss", `{"code": "lossy.key", "message": "m"}`),
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
	if len(results) != 7 {
		t.Fatalf("%d results, want 7", len(results))
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

// A file and a directory cannot share a path, in either order, so the
// model hears of it rather than the host failing to write.
func TestPathCollisionsAreRefused(t *testing.T) {
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("write_file", `{"path": "lib/shop.ex", "content": "x"}`),
			call("write_file", `{"path": "lib", "content": "x"}`),
			call("write_file", `{"path": "lib/shop.ex/order.ex", "content": "x"}`),
			call("write_file", `{"path": "lib/shop.exs", "content": "x"}`),
			call("write_file", `{"path": "LIB", "content": "x"}`),
			call("write_file", `{"path": "Lib/Shop.ex", "content": "x"}`),
			call("write_file", `{"path": "Lib/Shop.ex/order.ex", "content": "x"}`),
		}},
		{Role: yolo.Assistant, Stop: yolo.StopEnd, Text: "Done."},
	}}
	resp, err := (yolo.Backend{Provider: f}).Generate(context.Background(), request(t, order))
	if err != nil {
		t.Fatal(err)
	}
	var refused []bool
	for _, r := range f.seen[1].Messages[2].Results {
		refused = append(refused, r.Error)
	}
	if want := []bool{false, true, true, false, true, true, true}; !slices.Equal(refused, want) {
		t.Errorf("refused %v, want %v", refused, want)
	}
	var paths []string
	for _, file := range resp.GetFiles() {
		paths = append(paths, file.GetPath())
	}
	slices.Sort(paths)
	if want := []string{"lib/shop.ex", "lib/shop.exs"}; !slices.Equal(paths, want) {
		t.Errorf("files %v, want %v", paths, want)
	}
	if strings.Contains(diagnostics(resp), "error") {
		t.Errorf("diagnostics %s", diagnostics(resp))
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
	if len(f.seen) != 50 || !strings.Contains(diagnostics(resp), "still calling tools") {
		t.Errorf("sent %d turns; diagnostics %s", len(f.seen), diagnostics(resp))
	}
}

// The prompt carries the language, the instructions, the model as TDL,
// the directives chosen, every loss code, and the spec.
func TestPrompt(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`,
		`language("elixir")
  prompt("Ecto embedded schemas")
  model("claude-sonnet-5-5")
  effort("low")`, 1)
	f := &fixed{replies: []yolo.Message{{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
		call("write_file", `{"path": "shop.ex", "content": "x"}`),
	}}, {Role: yolo.Assistant, Stop: yolo.StopEnd}}}
	if _, err := (yolo.Backend{Provider: f}).Generate(context.Background(), request(t, src)); err != nil {
		t.Fatal(err)
	}
	c := f.seen[0]
	if c.Model != "claude-sonnet-5-5" || c.Effort != "low" {
		t.Errorf("model %q, effort %q", c.Model, c.Effort)
	}
	first := c.Messages[0].Text
	for _, want := range []string{"Write this model in elixir.", "Ecto embedded schemas", "type Order: Entity {", "target yolo for shop {"} {
		if !strings.Contains(first, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, first)
		}
	}
	for _, code := range emit.LossCodes {
		_, rest, ok := strings.Cut(c.System, "- "+code+": ")
		if line, _, _ := strings.Cut(rest, "\n"); !ok || strings.TrimSpace(line) == "" {
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
	if f.seen[0].Model != yolo.DefaultModel || f.seen[0].Effort != yolo.DefaultEffort {
		t.Errorf("model %q, effort %q", f.seen[0].Model, f.seen[0].Effort)
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

// A replay refuses a conversation whose earlier messages changed after
// they were answered.
func TestReplayRefusesAChangedHistory(t *testing.T) {
	dir := filepath.Join(cases, "order")
	r, err := yolo.LoadReplay(dir)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := yolo.Backend{Provider: &rewriting{Provider: r}}.Generate(context.Background(), request(t, Source(t, dir)))
	if err != nil {
		t.Fatal(err)
	}
	if got := diagnostics(resp); !strings.Contains(got, "changed the conversation") {
		t.Errorf("diagnostics: %s", got)
	}
}

// rewriting changes the first message before every turn but the first.
type rewriting struct {
	yolo.Provider
	turns int
}

func (r *rewriting) Send(ctx context.Context, c *yolo.Conversation) (yolo.Message, error) {
	if r.turns++; r.turns > 1 {
		c.Messages[0].Text += "changed"
	}
	return r.Provider.Send(ctx, c)
}

// The model is not shown what a dependency generates, so each extern warns.
func TestExternsWarn(t *testing.T) {
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Stop: yolo.StopTool, Calls: []yolo.Call{
			call("write_file", `{"path": "shop.ex", "content": "x"}`),
		}},
		{Role: yolo.Assistant, Stop: yolo.StopEnd},
	}}
	req := request(t, order)
	req.Model.Externs = append(req.Model.Externs, &ir.Extern{
		Package: "money", Name: "Money", Position: &ir.Position{Line: 3, Column: 7},
	})
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	got := diagnostics(resp)
	if !strings.Contains(got, "3:7 warning "+emit.LossUnsupported+": money.Money comes from another file") {
		t.Errorf("diagnostics: %s", got)
	}
	if len(resp.GetFiles()) != 1 {
		t.Errorf("files: %v", resp.GetFiles())
	}
}

// checked lowers src as a file beside a tdl.toml holding manifest, with
// the output in a directory of its own.
func checked(t *testing.T, manifest, src string) *plugin.Request {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tdl.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	model := lowerAt(t, filepath.Join(dir, "source.tdl"), src)
	req, diags := analysis.Request(model, yolo.Backend{}.Describe(), yolo.Name, filepath.Join(dir, "out"))
	if len(diags) > 0 {
		t.Fatalf("request: %v", diags)
	}
	return req
}

// What no try can fix is an error before anything is sent: a tries below
// one, or a check name tdl.toml does not declare.
func TestCheckSetupErrors(t *testing.T) {
	for name, tc := range map[string]struct{ manifest, directives, want string }{
		"no tries":   {"", `tries(0)`, "at least 1"},
		"undeclared": {"[check]\n", `check("mix")`, `"mix"`},
		"no table":   {"", `check("mix")`, `"mix"`},
	} {
		t.Run(name, func(t *testing.T) {
			src := strings.Replace(order, `language("elixir")`, `language("elixir")`+"\n  "+tc.directives, 1)
			f := &fixed{}
			resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), checked(t, tc.manifest, src))
			if err != nil {
				t.Fatal(err)
			}
			if len(f.seen) != 0 || len(resp.GetFiles()) != 0 {
				t.Errorf("sent %d turns and wrote %d files", len(f.seen), len(resp.GetFiles()))
			}
			if got := diagnostics(resp); !strings.Contains(got, "error") || !strings.Contains(got, tc.want) {
				t.Errorf("diagnostics: %s", got)
			}
		})
	}
}

// A check command that is not found is the project's to fix, not the
// model's, so it is an error after the first answer, with no retry.
func TestMissingCheckCommandIsNotRetried(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`, `language("elixir")`+"\n  check(\"mix\")", 1)
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Calls: []yolo.Call{call("write_file", `{"path": "order.ex", "content": "x"}`)}, Stop: yolo.StopTool},
		{Role: yolo.Assistant, Text: "done", Stop: yolo.StopEnd},
	}}
	req := checked(t, "[check]\nmix = \"tdl-no-such-command --strict\"\n", src)
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seen) != 2 || len(resp.GetFiles()) != 0 {
		t.Errorf("sent %d turns and wrote %d files", len(f.seen), len(resp.GetFiles()))
	}
	if got := diagnostics(resp); !strings.Contains(got, "was not found") {
		t.Errorf("diagnostics: %s", got)
	}
}

// A dry run's files are compared, not written, so no check runs for one.
func TestDryRunRunsNoCheck(t *testing.T) {
	src := strings.Replace(order, `language("elixir")`, `language("elixir")`+"\n  check(\"mix\")", 1)
	f := &fixed{replies: []yolo.Message{
		{Role: yolo.Assistant, Calls: []yolo.Call{call("write_file", `{"path": "order.ex", "content": "x"}`)}, Stop: yolo.StopTool},
		{Role: yolo.Assistant, Text: "done", Stop: yolo.StopEnd},
	}}
	req := checked(t, "[check]\nmix = \"false\"\n", src)
	req.DryRun = true
	resp, err := yolo.Backend{Provider: f}.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFiles()) != 1 || diagnostics(resp) != "" {
		t.Errorf("wrote %d files: %s", len(resp.GetFiles()), diagnostics(resp))
	}
}
