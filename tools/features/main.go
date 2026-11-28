// Command features measures every backend against testdata/features and
// rewrites the tables markdown files embed between tdl:features markers.
//
// Run it from the module root:
//
//	go run ./tools/features
//
// Every tool an import runs, and every plugin, must be on PATH; -short
// leaves the cells needing them as they are committed.
package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/unstoppablemango/tdl/internal/features"
)

func main() {
	root := flag.String("root", ".", "the module root")
	short := flag.Bool("short", false, "keep the committed cells that need a tool or plugin on PATH")
	flag.Parse()

	if err := run(*root, *short); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, short bool) error {
	docs, err := features.Update(root, short)
	if err != nil {
		return err
	}
	for _, doc := range slices.Sorted(maps.Keys(docs)) {
		path := filepath.Join(root, doc)
		old, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(old) == string(docs[doc]) {
			continue
		}
		if err := os.WriteFile(path, docs[doc], 0o644); err != nil {
			return err
		}
	}
	return nil
}
