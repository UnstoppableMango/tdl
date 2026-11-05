package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/host"
)

// completion is what one __complete call offers.
type completion struct {
	values    []string
	descs     map[string]string
	directive cobra.ShellCompDirective
}

// complete runs the hidden __complete command as a shell does, with the
// word being completed last.
func complete(t *testing.T, args ...string) completion {
	t.Helper()
	root := newRootCmd()
	out, _ := captureCmd(root)
	root.SetArgs(append([]string{cobra.ShellCompRequestCmd}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}

	c := completion{descs: map[string]string{}}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	last := lines[len(lines)-1]
	d, err := strconv.Atoi(strings.TrimPrefix(last, ":"))
	if err != nil || !strings.HasPrefix(last, ":") {
		t.Fatalf("__complete %v ended without a directive: %q", args, out.String())
	}
	c.directive = cobra.ShellCompDirective(d)
	for _, line := range lines[:len(lines)-1] {
		value, desc, _ := strings.Cut(line, "\t")
		c.values = append(c.values, value)
		c.descs[value] = desc
	}
	return c
}

// fakePlugins puts tdl-gen-x, and a tdl-gen-go a built-in shadows, alone
// on PATH. Each records that it ran, which completion must never do.
func fakePlugins(t *testing.T) (dir, ran string) {
	t.Helper()
	dir = t.TempDir()
	ran = filepath.Join(t.TempDir(), "ran")
	script := "#!/bin/sh\ntouch " + ran + "\nexit 1\n"
	for _, name := range []string{"x", "go"} {
		if err := os.WriteFile(filepath.Join(dir, gen.CommandPrefix+name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Neither a directory nor a file that cannot run is a plugin.
	if err := os.Mkdir(filepath.Join(dir, gen.CommandPrefix+"dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, gen.CommandPrefix+"data"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return dir, ran
}

// models writes a.tdl, declaring a haskell block, and b.tdl, importing
// a.tdl and declaring a zod block, and a file that does not lower.
func models(t *testing.T) (a, b, bad string) {
	t.Helper()
	dir := t.TempDir()
	a, b, bad = filepath.Join(dir, "a.tdl"), filepath.Join(dir, "b.tdl"), filepath.Join(dir, "bad.tdl")
	files := map[string]string{
		a:   "package a\n\nprimitive string\n\ntype A { s: string }\n\ntarget haskell for a {\n  out(\"./hs\")\n}\n",
		b:   "package b\n\nimport \"a.tdl\" as a\n\ntype B { x: a.A }\n\ntarget zod for b { }\n",
		bad: "package bad\n\ntype C { x: Missing }\n\ntarget go for bad { }\n",
	}
	for path, src := range files {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return a, b, bad
}

// everyTarget is what request --target offers with fakePlugins on PATH.
func everyTarget() []string {
	return append(gen.BuiltinNames(), "x")
}

func sorted(s []string) []string {
	return slices.Sorted(slices.Values(s))
}

func TestCompleteRequestTarget(t *testing.T) {
	dir, ran := fakePlugins(t)

	c := complete(t, "request", "--target", "")
	if got, want := sorted(c.values), sorted(everyTarget()); !slices.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}
	if c.descs["go"] != "built in" {
		t.Errorf("go reads %q, want built in", c.descs["go"])
	}
	if want := filepath.Join(dir, "tdl-gen-x"); c.descs["x"] != want {
		t.Errorf("x reads %q, want %s", c.descs["x"], want)
	}
	if c.directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %d, want no file completion", c.directive)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("completion started a plugin")
	}

	if c := complete(t, "request", "--target", "has"); !slices.Equal(c.values, []string{"haskell"}) {
		t.Errorf("has completes to %v, want [haskell]", c.values)
	}
}

func TestCompleteImportFrom(t *testing.T) {
	_, ran := fakePlugins(t)

	c := complete(t, "import", "--from", "")
	if got, want := sorted(c.values), sorted(append(gen.ReverseNames(), "x")); !slices.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}
	if slices.Contains(c.values, "debug") {
		t.Error("debug does not import, yet is offered")
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("completion started a plugin")
	}
}

func TestCompleteGenTarget(t *testing.T) {
	_, ran := fakePlugins(t)
	a, b, bad := models(t)

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"own block only", []string{b}, []string{"zod"}},
		{"every file", []string{a, b}, []string{"haskell", "zod"}},
		{"no files", nil, everyTarget()},
		{"a file that does not lower", []string{a, bad}, everyTarget()},
		{"a missing file", []string{filepath.Join(t.TempDir(), "gone.tdl")}, everyTarget()},
		{"standard input", []string{"-"}, everyTarget()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := slices.Concat([]string{"gen"}, tc.args, []string{"--target", ""})
			c := complete(t, args...)
			if got, want := sorted(c.values), sorted(tc.want); !slices.Equal(got, want) {
				t.Errorf("values = %v, want %v", got, want)
			}
			if c.descs["zod"] != "" && c.descs["zod"] != "built in" {
				t.Errorf("zod reads %q, want built in", c.descs["zod"])
			}
		})
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("completion started a plugin")
	}
}

func TestCompleteTDLFiles(t *testing.T) {
	a, _, _ := models(t)
	for _, name := range []string{"ast", "check", "fmt", "gen", "ir", "play", "request", "tokens"} {
		t.Run(name, func(t *testing.T) {
			c := complete(t, name, "")
			// cobra also offers a required flag, such as request's --target.
			exts := slices.DeleteFunc(c.values, func(v string) bool { return strings.HasPrefix(v, "-") })
			if c.directive != cobra.ShellCompDirectiveFilterFileExt || !slices.Equal(exts, []string{"tdl"}) {
				t.Errorf("got %v with directive %d, want [tdl] filtered by extension", c.values, c.directive)
			}
		})
	}

	// gen takes many files; play takes one.
	if c := complete(t, "gen", a, ""); c.directive != cobra.ShellCompDirectiveFilterFileExt {
		t.Errorf("gen's second file: directive %d", c.directive)
	}
	if c := complete(t, "play", a, ""); c.directive != cobra.ShellCompDirectiveNoFileComp || len(c.values) != 0 {
		t.Errorf("play's second file: %v with directive %d, want nothing", c.values, c.directive)
	}
}

func TestCompleteAnyFile(t *testing.T) {
	cases := [][]string{
		{"import", ""},
		{"gen", "-o", ""},
		{"gen", "--make", ""},
		{"import", "-o", ""},
		{"ir", "--prelude", ""},
		{"lsp", "--prelude", ""},
	}
	for _, args := range cases {
		if c := complete(t, args...); c.directive != cobra.ShellCompDirectiveDefault || len(c.values) != 0 {
			t.Errorf("%v: %v with directive %d, want every file", args, c.values, c.directive)
		}
	}
	if c := complete(t, "import", "--package", ""); c.directive != cobra.ShellCompDirectiveNoFileComp || len(c.values) != 0 {
		t.Errorf("--package: %v with directive %d, want nothing", c.values, c.directive)
	}
}

func TestCompleteFixedValues(t *testing.T) {
	c := complete(t, "ir", "--format", "")
	if !slices.Equal(sorted(c.values), []string{"json", "text"}) || c.directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("--format: %v with directive %d", c.values, c.directive)
	}
}

func TestCompleteViews(t *testing.T) {
	commaList := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp

	c := complete(t, "play", "--views", "")
	if got, want := sorted(c.values), sorted(append([]string{"all"}, playViews...)); !slices.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}
	if c.directive != commaList {
		t.Errorf("directive = %d, want %d", c.directive, commaList)
	}

	// all is the first value only, and a view already given is not offered.
	c = complete(t, "play", "--views", "fmt,")
	if got, want := sorted(c.values), []string{"fmt,ast", "fmt,source", "fmt,stats", "fmt,tokens"}; !slices.Equal(got, want) {
		t.Errorf("after fmt, values = %v, want %v", got, want)
	}
	if c := complete(t, "play", "--views", "fmt,t"); !slices.Equal(c.values, []string{"fmt,tokens"}) {
		t.Errorf("fmt,t completes to %v", c.values)
	}
	if c := complete(t, "play", "--views", "all,"); len(c.values) != 0 {
		t.Errorf("after all, values = %v, want none", c.values)
	}
}

func TestCompleteAllowLossy(t *testing.T) {
	commaList := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	for _, name := range []string{"gen", "import", "request"} {
		t.Run(name, func(t *testing.T) {
			c := complete(t, name, "--allow-lossy", "")
			if !slices.Equal(c.values, host.LossCodes) || c.directive != commaList {
				t.Errorf("got %v with directive %d, want every loss code with %d", c.values, c.directive, commaList)
			}
			for v, d := range c.descs {
				if d != "" {
					t.Errorf("%s reads %q, want no description", v, d)
				}
			}

			c = complete(t, name, "--allow-lossy", "lossy.order,lossy.n")
			if want := []string{"lossy.order,lossy.newtype", "lossy.order,lossy.name", "lossy.order,lossy.number"}; !slices.Equal(c.values, want) {
				t.Errorf("after lossy.order, values = %v, want %v", c.values, want)
			}
		})
	}
}
