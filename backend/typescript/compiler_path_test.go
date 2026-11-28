package typescript_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/unstoppablemango/tdl/backend/typescript"
)

// Compiler takes the compiler API from tsc's package on PATH, and passes
// over a tsc without one, as TypeScript 7's native binary is, for a
// typescript package on NODE_PATH.
func TestCompilerFindsTheAPI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tools are shell scripts, which PATHEXT does not find")
	}
	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	// macOS's temporary directory is behind a symlink, which Compiler
	// resolves for tsc, so the expected paths are resolved too.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	write(filepath.Join(bin, "node"), "#!/bin/sh\n", 0o755)

	// A package holding the API, its tsc linked onto PATH.
	ts5 := filepath.Join(root, "ts5", "typescript")
	write(filepath.Join(ts5, "bin", "tsc"), "#!/bin/sh\n", 0o755)
	write(filepath.Join(ts5, "lib", "typescript.js"), "", 0o644)
	// A native tsc, as TypeScript 7's, with no API beside it.
	ts7 := filepath.Join(root, "ts7")
	write(filepath.Join(ts7, "lib", "typescript", "tsc"), "#!/bin/sh\n", 0o755)

	for name, c := range map[string]struct{ tsc, nodePath, want string }{
		"tsc on PATH":                  {tsc: filepath.Join(ts5, "bin", "tsc"), want: ts5},
		"native tsc, API on NODE_PATH": {tsc: filepath.Join(ts7, "lib", "typescript", "tsc"), nodePath: filepath.Dir(ts5), want: ts5},
		"native tsc alone":             {tsc: filepath.Join(ts7, "lib", "typescript", "tsc")},
		"no tsc, API on NODE_PATH":     {nodePath: filepath.Dir(ts5), want: ts5},
	} {
		t.Run(name, func(t *testing.T) {
			path := t.TempDir()
			if err := os.Symlink(filepath.Join(bin, "node"), filepath.Join(path, "node")); err != nil {
				t.Fatal(err)
			}
			if c.tsc != "" {
				if err := os.Symlink(c.tsc, filepath.Join(path, "tsc")); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", path)
			t.Setenv("NODE_PATH", c.nodePath)
			_, pkg, err := typescript.Compiler()
			if c.want == "" {
				if err == nil {
					t.Errorf("Compiler found %s, want ErrNoCompiler", pkg)
				}
				return
			}
			if err != nil || pkg != c.want {
				t.Errorf("Compiler = %q, %v, want %q", pkg, err, c.want)
			}
		})
	}
}
