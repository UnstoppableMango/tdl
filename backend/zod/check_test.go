package zod_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/plugin"
)

// The tests type check every response with tsc and run it under node with
// the zod package installed. Without tsc, node, or zod, or under -short,
// they check the text alone; checks.gen-zod and the devShell provide all
// three, zod through NODE_PATH.

// tscFlags are the compiler options every check uses: strict, with
// exactOptionalPropertyTypes so `name?: T` refuses an explicit undefined
// as the schema does, and Node's module resolution, which the generated
// imports follow.
var tscFlags = []string{
	"--strict", "--exactOptionalPropertyTypes",
	"--module", "nodenext", "--moduleResolution", "nodenext", "--target", "es2022",
}

// toolchain is tsc, node, and the zod package's directory, or why one is
// missing.
func toolchain() (tsc, node, zod string, why string) {
	if testing.Short() {
		return "", "", "", "-short"
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		return "", "", "", "tsc is not on PATH"
	}
	node, err = exec.LookPath("node")
	if err != nil {
		return "", "", "", "node is not on PATH"
	}
	for _, dir := range filepath.SplitList(os.Getenv("NODE_PATH")) {
		if _, err := os.Stat(filepath.Join(dir, "zod", "package.json")); err == nil {
			return tsc, node, filepath.Join(dir, "zod"), ""
		}
	}
	return "", "", "", "no zod package on NODE_PATH"
}

// project writes files into a directory that resolves zod, as an ES
// module package, and returns it.
func project(t *testing.T, zod string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(zod, filepath.Join(dir, "node_modules", "zod")); err != nil {
		t.Fatal(err)
	}
	files["package.json"] = `{"type": "module"}`
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// sources is the path of every .ts file in files.
func sources(dir string, files map[string]string) []string {
	var out []string
	for name := range files {
		if strings.HasSuffix(name, ".ts") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}

// responseFiles is the files of the responses by path.
func responseFiles(resps ...*plugin.Response) map[string]string {
	files := map[string]string{}
	for _, r := range resps {
		for _, f := range r.GetFiles() {
			files[filepath.Base(f.GetPath())] = string(f.GetContent())
		}
	}
	return files
}

// typecheck runs tsc over the responses and any extra files, and fails
// the test when it rejects them.
func typecheck(t *testing.T, extra map[string]string, resps ...*plugin.Response) {
	t.Helper()
	tsc, _, zod, why := toolchain()
	if why != "" {
		t.Logf("not type checked: %s", why)
		return
	}
	files := responseFiles(resps...)
	for k, v := range extra {
		files[k] = v
	}
	dir := project(t, zod, files)
	args := append(append([]string{"--noEmit"}, tscFlags...), sources(dir, files)...)
	if out, err := exec.Command(tsc, args...).CombinedOutput(); err != nil {
		t.Errorf("tsc rejects the output: %v\n%s\n%s", err, out, dump(files))
	}
}

// rejects runs tsc over the responses and a snippet that must not type
// check, and fails the test when tsc accepts it or rejects the responses
// alone.
func rejects(t *testing.T, snippet string, resps ...*plugin.Response) {
	t.Helper()
	tsc, _, zod, why := toolchain()
	if why != "" {
		t.Logf("not type checked: %s", why)
		return
	}
	files := responseFiles(resps...)
	files["reject.ts"] = snippet
	dir := project(t, zod, files)
	args := append(append([]string{"--noEmit"}, tscFlags...), sources(dir, files)...)
	if out, err := exec.Command(tsc, args...).CombinedOutput(); err == nil {
		t.Errorf("tsc accepts what it must reject:\n%s", snippet)
	} else if !strings.Contains(string(out), "reject.ts") {
		t.Errorf("tsc rejects the output itself:\n%s\n%s", out, dump(files))
	}
}

// harness is the module a parse test imports: accepts and refuses record
// each schema that disagrees, and done exits with them. process is
// declared here, since node's types may not be installed.
const harness = `declare const process: { exit(code: number): never };
type Schema = { safeParse(v: unknown): { success: boolean } };
const failures: string[] = [];
export function accepts(what: string, s: Schema, json: string): void {
  if (!s.safeParse(JSON.parse(json)).success) failures.push("refuses " + what + ": " + json);
}
export function refuses(what: string, s: Schema, json: string): void {
  if (s.safeParse(JSON.parse(json)).success) failures.push("accepts " + what + ": " + json);
}
export function done(): void {
  if (failures.length > 0) {
    console.log(failures.join("\n"));
    process.exit(1);
  }
}
`

// parse compiles the responses with a script and runs it under node. The
// script calls accepts and refuses, and imports the generated module by
// its path with a .js extension.
func parse(t *testing.T, script string, resps ...*plugin.Response) {
	t.Helper()
	tsc, node, zod, why := toolchain()
	if why != "" {
		t.Logf("not run: %s", why)
		return
	}
	files := responseFiles(resps...)
	files["harness.ts"] = harness
	files["main.ts"] = "import { accepts, refuses, done } from \"./harness.js\";\n" + script + "\ndone();\n"
	dir := project(t, zod, files)
	args := append(append([]string{"--outDir", dir}, tscFlags...), sources(dir, files)...)
	if out, err := exec.Command(tsc, args...).CombinedOutput(); err != nil {
		t.Fatalf("tsc rejects the parse test: %v\n%s\n%s", err, out, dump(files))
	}
	if out, err := exec.Command(node, filepath.Join(dir, "main.js")).CombinedOutput(); err != nil {
		t.Errorf("node: %v\n%s\n%s", err, out, dump(files))
	}
}

func dump(files map[string]string) string {
	var b strings.Builder
	for name, content := range files {
		if strings.HasSuffix(name, ".ts") {
			b.WriteString("==> " + name + " <==\n" + content + "\n")
		}
	}
	return b.String()
}
