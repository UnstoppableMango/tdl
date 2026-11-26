// Command skills writes the skill library's generated references: the docs
// and prelude each skill carries, and the release they were copied from.
//
// Run it from the module root:
//
//	go run ./tools/skills
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/skills"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	for p, data := range skills.References() {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
