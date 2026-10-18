// Command features measures every backend against testdata/features and
// rewrites the tables markdown files embed between tdl:features markers.
//
// Run it from the module root:
//
//	go run ./tools/features
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
	flag.Parse()

	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	docs, err := features.Update(root, false)
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
