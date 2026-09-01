# Implementing the tree-sitter grammar

An implementation plan for [treesitter.md](treesitter.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds the derivation: the annotations, the reader, the emitter, and the checks that hold the two grammars together.

It does not build editor support.
Highlighting queries get a first pass so the grammar is useful to look at, and the four editor integrations in [backlog.md](../backlog.md) stay there.

It does not touch the reference implementation.
`lex` gained a table describing itself and nothing about scanning changed; `parser` is not read by any of this.

## Layout

```text
docs/grammar.ebnf     # annotated, still the canonical grammar
internal/ebnf/        # golang.org/x/exp/ebnf plus the annotations, private
internal/treesitter/  # model to grammar.js, private
tools/treesitter/     # main, run with go run
tree-sitter/          # grammar.js, package.json, queries, generated src
```

`tools/` is a build tool rather than something shipped, which is why it is not under `cmd/`.

## Testing

The reader and the emitter are Go, and get Go tests.

The grammar gets the corpus, run by `tree-sitter parse` rather than by Go, so a check the reference implementation passes is one a derived parser has to pass in the same terms.

`grammar.js` is committed and regenerated, and the diff is read rather than trusted, the same way `ir.pb.go` and the ir goldens are.

## Phase 1: the lexical tables

`lex` states its own facts: `Keywords`, `Punctuation`, `Lookup`, `Spelling`, and `Pattern`, with tests holding the patterns to the lexer over table cases and the whole corpus.

`docs/notation.ebnf` describes the notation in itself, and `internal/ebnf` lints both grammar files against it.

Done.

## Phase 2: annotate the grammar

`docs/grammar.ebnf` gains the file-level directives and the production annotations, and nothing else changes.
No generator reads them yet, so this phase is checked by reading: the file still explains itself to a person, and every annotation sits next to the prose comment that already made the same point.

Done when every terminal has a `token` binding, every ambiguity the file describes in prose has a `conflict` or a `prec`, and the plumbing productions carry `hidden` or `inline`.

Done.
The `conflict` and `prec` entries are the file's prose read as annotations and are provisional until phase 5 generates a parser that agrees or does not.

## Phase 3: the annotations reach the reader

`internal/ebnf` already reads `docs/grammar.ebnf`: `golang.org/x/exp/ebnf` parses the notation and checks reachability, and the package adds the check no library makes, that a quoted terminal is text `lex` turns into exactly one token.

What is missing is the annotations, which the library drops with the rest of the comments.
Scanning them separately and attaching each to the production or the file it belongs to is the whole of this phase.

Done when every annotation in phase 2 is readable from Go, and an annotation naming a production that does not exist, or a terminal with no `token` binding, is reported with the line that caused it.

## Phase 4: the emitter

`internal/treesitter` walks the model and writes `grammar.js`: `seq`, `choice`, `optional`, `repeat`, the `extras`, `conflicts`, `externals`, `inline`, and `word` entries, and a rule per terminal built from `lex.Pattern` or from the spelling.

Output is deterministic, since a nondeterministic generator makes the regeneration check useless.

Done when `tools/treesitter` writes a `grammar.js` that `tree-sitter generate` accepts, and running it twice produces the same bytes.

## Phase 5: the conformance corpus

`tree-sitter/package.json`, the generated parser, and the corpus run.

Every `testdata/conformance/*/source.tdl` parses with no ERROR node.
This is where the ambiguities become real, and where the `conflict` and `prec` annotations of phase 2 are either right or revised.

Done when the whole conformance corpus parses clean.

## Phase 6: the external scanner

`scanner.c` for `regex_lit`, which the built-in lexer cannot separate from unit division without knowing what the parser wants.

The reference implementation solves this with `RescanRegexAt`, and the shape of the problem carries over: the scanner produces a regex only where the grammar allows one.

Done when the constraint cases parse, `matches(/^[^@]+@[^@]+$/)` and `unit N = kg * m / s^2` both read correctly, and `testdata/invalid/*/source.tdl` each produce an ERROR node.

## Phase 7: wiring

`command make generate-treesitter`, a CI job running it followed by `git diff --exit-code`, and a first `queries/highlights.scm`.

`AGENTS.md` gains the derivation, and `docs/backlog.md` loses the tree-sitter entry it had before this plan existed.

Done when a production added to `docs/grammar.ebnf` without regenerating fails CI.

## Not in this plan

- Editor packaging. Vim, VS Code, JetBrains, and Emacs stay in the backlog, and each wants the language server more than it wants this.
- The language server. Separate work with its own dependencies, and nothing here blocks it.
- `union`. Reserved by the lexer, unimplemented by the parser, and there is nothing to derive until that changes.
- Publishing to npm or to a grammar registry, which is a distribution question and not a correctness one.
