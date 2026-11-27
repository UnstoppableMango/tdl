package cli

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updatePortable = flag.Bool("update", false, "rewrite testdata/gen/smoke/output.golden")

// smokeTargets are the smoke model's target blocks that tdl serves
// itself; csharp and mermaid are plugins found on PATH.
var smokeTargets = []string{
	"graphql", "haskell", "jsonschema", "openapi", "protobuf",
	"salesforce", "smithy", "thrift", "typescript", "zod",
}

// TestSmokeOutputIsPortable generates the smoke model for every built-in
// target and holds each file's slash-separated path and digest to
// output.golden, so every OS writes the same bytes to the same paths.
// go test ./internal/cli -run TestSmokeOutputIsPortable -update rewrites
// it.
func TestSmokeOutputIsPortable(t *testing.T) {
	const source = "../../testdata/gen/smoke/source.tdl"
	golden := filepath.Join("..", "..", "testdata", "gen", "smoke", "output.golden")
	root := t.TempDir()
	var b strings.Builder
	for _, target := range smokeTargets {
		out := filepath.Join(root, target)
		if _, errOut, err := run(t, newGenCmd(), "--target", target, "-o", out, source); err != nil {
			t.Fatalf("gen --target %s: %v\n%s", target, err, errOut)
		}
		err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			fmt.Fprintf(&b, "%x  %s\n", sha256.Sum256(data), filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if *updatePortable {
		if err := os.WriteFile(golden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v; run with -update", err)
	}
	if got := b.String(); got != string(want) {
		t.Errorf("generated output differs from output.golden\ngot:\n%s\nwant:\n%s", got, want)
	}
}
