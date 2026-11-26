package skills_test

import (
	"bufio"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/skills"
	"github.com/unstoppablemango/tdl/parser"
)

const root = "../.."

// Every generated reference is what `make skills` writes now.
func TestReferencesAreCurrent(t *testing.T) {
	refs := skills.References()
	for _, p := range slices.Sorted(maps.Keys(refs)) {
		committed, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Errorf("%v; run make skills", err)
			continue
		}
		if string(committed) != string(refs[p]) {
			t.Errorf("%s is stale; run make skills and read the diff", p)
		}
	}
	for _, dir := range skillDirs(t) {
		err := filepath.WalkDir(filepath.Join(dir, "references"), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if _, ok := refs[filepath.ToSlash(rel)]; !ok && err == nil {
				t.Errorf("%s is no generated reference; delete it or list it in internal/skills", rel)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// release-please bumps every skill's version line, or a release would leave
// it stale.
func TestReleasePleaseBumpsVersions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root, "release-please-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range skills.Names() {
		file := path.Join(skills.Dir, name, skills.VersionFile)
		if !strings.Contains(string(data), `"`+file+`"`) {
			t.Errorf("release-please-config.json's extra-files does not list %s", file)
		}
	}
}

// Every skill's frontmatter names it after its folder, under the tdl-
// prefix, and describes it, with nothing an agent other than Claude would
// not read.
func TestFrontmatter(t *testing.T) {
	for _, dir := range skillDirs(t) {
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			fields := frontmatter(t, filepath.Join(dir, "SKILL.md"))
			if fields["name"] != name {
				t.Errorf("name is %q, want the folder's name %q", fields["name"], name)
			}
			if !strings.HasPrefix(name, "tdl-") {
				t.Errorf("%s does not start with tdl-", name)
			}
			if fields["description"] == "" {
				t.Error("no description")
			}
			for k := range fields {
				if k != "name" && k != "description" {
					t.Errorf("frontmatter holds %s; only name and description are allowed", k)
				}
			}
		})
	}
}

// Every skill with generated references is a folder, and every folder's
// references are generated.
func TestEverySkillIsListed(t *testing.T) {
	var names []string
	for _, dir := range skillDirs(t) {
		names = append(names, filepath.Base(dir))
	}
	if !slices.Equal(names, skills.Names()) {
		t.Errorf("skills/ holds %v, internal/skills lists %v", names, skills.Names())
	}
}

// Every ```tdl block in a hand-written file is a whole file that parses,
// lowers with no diagnostic, and is canonical.
func TestSnippets(t *testing.T) {
	generated := skills.References()
	found := 0
	for _, dir := range skillDirs(t) {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			if _, ok := generated[filepath.ToSlash(rel)]; ok {
				return nil
			}
			for _, s := range snippets(t, p) {
				found++
				t.Run(s.name, func(t *testing.T) { checkSnippet(t, s) })
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if found == 0 {
		t.Fatal("no tdl blocks found")
	}
}

// CheckSnippet fails a broken block, so a test of the test can hold it to
// failing one.
func TestBrokenSnippetFails(t *testing.T) {
	for _, src := range []string{
		"type Order {\n",               // does not parse
		"type Order { id: Missing }\n", // does not lower
		"type Order {id:string}\n",     // not canonical
	} {
		if problem(snippet{name: "broken", src: src}) == "" {
			t.Errorf("%q passed", src)
		}
	}
}

type snippet struct {
	name string
	src  string
}

func checkSnippet(t *testing.T, s snippet) {
	t.Helper()
	if msg := problem(s); msg != "" {
		t.Error(msg)
	}
}

// problem is why a block is not a whole canonical file, or empty.
func problem(s snippet) string {
	file, err := parser.Parse(s.name, strings.NewReader(s.src))
	if err != nil {
		return err.Error()
	}
	if _, diags := sema.Lower(file); len(diags) > 0 {
		return diags.Error()
	}
	if got := ast.Fprint(file); got != s.src {
		return "not canonical; tdl fmt prints\n" + got
	}
	return ""
}

// snippets reads every ```tdl block in a markdown file, named by its line.
func snippets(t *testing.T, p string) []snippet {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var out []snippet
	var cur *strings.Builder
	start := 0
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		switch {
		case cur == nil && strings.TrimSpace(text) == "```tdl":
			cur, start = &strings.Builder{}, line
		case cur != nil && strings.TrimSpace(text) == "```":
			out = append(out, snippet{name: filepath.Base(filepath.Dir(p)) + ":" + strconv.Itoa(start), src: cur.String()})
			cur = nil
		case cur != nil:
			cur.WriteString(text + "\n")
		}
	}
	if cur != nil {
		t.Errorf("%s:%d: unterminated tdl block", p, start)
	}
	return out
}

func frontmatter(t *testing.T, p string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	head, ok := strings.CutPrefix(string(data), "---\n")
	if !ok {
		t.Fatal("SKILL.md does not open with frontmatter")
	}
	head, _, ok = strings.Cut(head, "\n---\n")
	if !ok {
		t.Fatal("frontmatter is not closed")
	}
	fields := map[string]string{}
	for line := range strings.SplitSeq(head, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			t.Errorf("frontmatter line %q is not a key and a value", line)
			continue
		}
		fields[k] = strings.TrimSpace(v)
	}
	return fields
}

func skillDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, skills.Dir))
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, skills.Dir, e.Name()))
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no skills")
	}
	return dirs
}
