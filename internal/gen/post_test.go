package gen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func postBlock(names ...string) *ir.TargetBlock {
	var args []*ir.Literal
	for _, n := range names {
		args = append(args, &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: n})
	}
	return &ir.TargetBlock{
		Meta:       &ir.Meta{Name: "ts"},
		Directives: []*ir.Directive{{Name: "post", Target: "ts", Args: args}},
	}
}

func TestResolvePost(t *testing.T) {
	cfg := &config.Config{
		Path: filepath.Join("proj", config.FileName),
		Post: map[string]string{"up": "tr a-z A-Z", "rev": "rev", "bad": "tr 'a"},
	}

	post, err := ResolvePost(postBlock("up", "rev"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if post.Dir != "proj" || len(post.Commands) != 2 || post.Commands[0].Name != "up" || post.Commands[1].Name != "rev" {
		t.Errorf("post = %+v", post)
	}

	if post, err := ResolvePost(&ir.TargetBlock{Meta: &ir.Meta{Name: "ts"}}, cfg); post != nil || err != nil {
		t.Errorf("a block without post resolved to %+v, %v", post, err)
	}

	errs := []struct {
		name  string
		block *ir.TargetBlock
		cfg   *config.Config
		want  []string
	}{
		{"undeclared", postBlock("up", "biome"), cfg, []string{"target ts", `"biome"`, "[post]", `biome = "<command>"`}},
		{"no tdl.toml", postBlock("up"), &config.Config{}, []string{"target ts", `"up"`, "no tdl.toml"}},
		{"malformed", postBlock("bad"), cfg, []string{"target ts", "post.bad", "unterminated"}},
	}
	for _, tc := range errs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolvePost(tc.block, tc.cfg)
			if err == nil {
				t.Fatal("resolved")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("err = %v, want it to contain %s", err, w)
				}
			}
		})
	}
}

// Commands run in order over every file, from the tdl.toml directory, with
// {file} the destination relative to it.
func TestPostApply(t *testing.T) {
	dir := t.TempDir()
	// The first command reads here from its working directory, which shows
	// where it ran without comparing paths, which Windows spells another way.
	if err := os.WriteFile(filepath.Join(dir, "here"), []byte("home\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	post := &Post{Dir: dir, Commands: []host.Command{
		{Name: "tag", Args: []string{"sh", "-c", `cat; echo "$1 $(cat here)"`, "sh", "{file}"}},
		{Name: "up", Args: []string{"tr", "a-z", "A-Z"}},
	}}
	files := []*plugin.File{
		{Path: "a.ts", Content: []byte("one\n")},
		{Path: "nested/b.ts", Content: []byte("two\n")},
	}

	got, err := post.Apply(context.Background(), filepath.Join(dir, "gen"), files)
	if err != nil {
		t.Fatal(err)
	}
	// The second command upper-cases what the first appended, so the order
	// shows.
	up := strings.ToUpper
	want := []string{
		"ONE\n" + up(filepath.Join("gen", "a.ts")) + " HOME\n",
		"TWO\n" + up(filepath.Join("gen", "nested", "b.ts")) + " HOME\n",
	}
	for i, f := range got {
		if f.GetPath() != files[i].GetPath() || string(f.GetContent()) != want[i] {
			t.Errorf("file %d = %s %q, want %s %q", i, f.GetPath(), f.GetContent(), files[i].GetPath(), want[i])
		}
	}
	if string(files[0].GetContent()) != "one\n" {
		t.Error("Apply changed the response's files")
	}
}

func TestPostApplyErrors(t *testing.T) {
	dir := t.TempDir()
	files := []*plugin.File{{Path: "a.ts", Content: []byte("x")}}
	script := filepath.Join(dir, "missing.sh")

	tests := []struct {
		name  string
		args  []string
		files []*plugin.File
		want  []string
	}{
		{"exits non-zero", []string{"sh", "-c", "echo broken >&2; exit 3"}, files, []string{"post command c failed on a.ts", "exit status 3", "broken"}},
		{"not on PATH", []string{"tdl-no-such-command"}, files, []string{"post command c on a.ts", "tdl-no-such-command was not found"}},
		{"no such file", []string{script}, files, []string{"post command c", script + " was not found"}},
		{"escaping path", []string{"cat"}, []*plugin.File{{Path: "../x.ts"}}, []string{"outside the output directory"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			post := &Post{Dir: dir, Commands: []host.Command{{Name: "c", Args: tc.args}}}
			_, err := post.Apply(context.Background(), dir, tc.files)
			if err == nil {
				t.Fatal("applied")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("err = %v, want it to contain %q", err, w)
				}
			}
		})
	}
}

// A nil Post leaves the files alone.
func TestNilPost(t *testing.T) {
	files := []*plugin.File{{Path: "a", Content: []byte("a")}}
	got, err := (*Post)(nil).Apply(context.Background(), os.TempDir(), files)
	if err != nil || len(got) != 1 || got[0] != files[0] {
		t.Errorf("got %v, %v", got, err)
	}
}
