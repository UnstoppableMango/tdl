package typescript

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindPackage lays out each way tsc is installed and finds the
// typescript package, holding the compiler API, from the tsc PATH would name.
func TestFindPackage(t *testing.T) {
	tests := []struct {
		name string
		// tsc is the compiler's path, and pkg the package's, under the root.
		tsc, pkg string
	}{
		{"package bin", "typescript/bin/tsc", "typescript"},
		{"nix prefix", "prefix/bin/tsc", "prefix/lib/node_modules/typescript"},
		{"npm global on windows", "npm/tsc.cmd", "npm/node_modules/typescript"},
		{"npm global powershell shim", "npm/tsc.ps1", "npm/node_modules/typescript"},
		{"npm project on windows", "project/node_modules/.bin/tsc.cmd", "project/node_modules/typescript"},
		{"uppercase extension", "npm/TSC.CMD", "npm/node_modules/typescript"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tsc := filepath.Join(root, filepath.FromSlash(tt.tsc))
			pkg := filepath.Join(root, filepath.FromSlash(tt.pkg))
			for _, f := range []string{tsc, filepath.Join(pkg, "lib", "typescript.js")} {
				if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(f, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, ok := findPackage(packageDirs(tsc))
			if !ok || got != pkg {
				t.Errorf("findPackage(%s) = %q, %v; want %q", tt.tsc, got, ok, pkg)
			}
		})
	}
}

// TestFindPackageWithoutOne refuses a tsc with no package beside it.
func TestFindPackageWithoutOne(t *testing.T) {
	tsc := filepath.Join(t.TempDir(), "npm", "tsc.cmd")
	if got, ok := findPackage(packageDirs(tsc)); ok {
		t.Errorf("findPackage found %q in an empty layout", got)
	}
}
