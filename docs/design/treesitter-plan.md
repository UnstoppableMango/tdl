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

Done.
`ebnf.Read` returns the grammar and an `Annotations` beside it; `Lint` is the same call with the grammar thrown away.
A `token` binding resolves to the pattern rather than the symbol name, so a caller never has to know `lex` to use one.

## Phase 4: the emitter

`internal/treesitter` walks the model and writes `grammar.js`: `seq`, `choice`, `optional`, `repeat`, the `extras`, `conflicts`, `externals`, `inline`, and `word` entries, and a rule per terminal built from `lex.Pattern` or from the spelling.

Output is deterministic, since a nondeterministic generator makes the regeneration check useless.

Done when `tools/treesitter` writes a `grammar.js` that `tree-sitter generate` accepts, and running it twice produces the same bytes.

Done.
Rule names are snake_case, the spelling every published tree-sitter grammar uses, and a hidden production gains the leading underscore; two productions that would claim one rule name are an error naming both.
Rules are emitted in the order the productions are written, so the diff against `docs/grammar.ebnf` is readable; nothing iterates the `Grammar` map.
The emitter substitutes an `inline` production itself rather than emitting tree-sitter's `inline` array, which refuses a rule that is a single token, as `FieldRel` is.
A character range and a bad node are errors rather than a guess, since the notation admits both and this grammar uses neither.

Generating a parser is what settled the provisional annotations of phase 2, one phase earlier than expected: `conflict Field KeyRequirement` is really `FieldMod KeyRequirement`, `conflict UnitExpr TypeRef` is really `DottedIdent UnitTerm`, `conflict Path Directive` is a conflict the generator resolves on its own and is gone, and a field followed by `where` needs `conflict Field`, a set of one, which `internal/ebnf` did not accept before.
`prec.left` on `UnitExpr` and `prec.right` on `Kind` were right.

The generated parser is not committed and `tree-sitter/.gitignore` says so, because whether it should be is a question for the phase that runs it.
`tree-sitter generate` warns that there is no `tree-sitter.json` and falls back to ABI 14; the manifest arrives with `package.json` in phase 5.

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
- Publishing to npm or to a grammar registry, which is a distribution question and not a correctness one.
