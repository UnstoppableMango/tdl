// Command treesitter derives tree-sitter/grammar.js from docs/grammar.ebnf.
//
// It is a build tool rather than something shipped, which is why it lives
// under tools/ and not cmd/. Run it from the module root:
//
//	go run ./tools/treesitter
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/treesitter"
)

func main() {
	in := flag.String("in", "docs/grammar.ebnf", "the annotated grammar to read")
	out := flag.String("out", "tree-sitter/grammar.js", "the grammar.js to write")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	file, err := ebnf.ReadFile(in, ebnf.GrammarOptions)
	if err != nil {
		return err
	}
	js, err := treesitter.Emit(file)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	return os.WriteFile(out, js, 0o644)
}
