// Command textmate derives the VS Code TextMate grammar from
// docs/grammar.ebnf.
//
// It is a build tool rather than something shipped, which is why it lives
// under tools/ and not cmd/. Run it from the module root:
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
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}

	file, errs := ebnf.Read(in, string(src), ebnf.GrammarOptions)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s: %d problems", in, len(errs))
	}

	json, err := textmate.Emit(file)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}

	return os.WriteFile(out, json, 0o644)
}
