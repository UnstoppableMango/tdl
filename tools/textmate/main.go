// Command textmate derives the VS Code TextMate grammar, and the
// sublime-syntax bat reads, from docs/grammar.ebnf.
//
// Run it from the module root:
//
//	go run ./tools/textmate
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/textmate"
)

func main() {
	in := flag.String("in", "docs/grammar.ebnf", "the annotated grammar to read")
	out := flag.String("out", "editors/vscode/syntaxes/tdl.tmLanguage.json", "the tmLanguage to write")
	sublime := flag.String("sublime", "editors/bat/tdl.sublime-syntax", "the sublime-syntax to write")
	flag.Parse()

	if err := run(*in, *out, *sublime); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in, out, sublime string) error {
	file, err := ebnf.ReadFile(in, ebnf.GrammarOptions)
	if err != nil {
		return err
	}
	json, err := textmate.Emit(file)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	if err := os.WriteFile(out, json, 0o644); err != nil {
		return err
	}
	syntax, err := textmate.EmitSublime(file)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	return os.WriteFile(sublime, syntax, 0o644)
}
