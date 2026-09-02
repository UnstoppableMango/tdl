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

Done.
`tree-sitter.json` is the manifest rather than `package.json`: the CLI reads the former, and npm packaging is not in this plan.
The generated parser is committed, the way `ir/ir.pb.go` is, so an editor or a `tree-sitter parse` gets a buildable grammar without the CLI, and the regeneration diff of phase 7 covers `parser.c` as well as `grammar.js`.

`tree-sitter/corpus.sh` runs the corpus, and `command make test-treesitter` runs the script; `command make treesitter` regenerates both committed artifacts.
The script carries a deferred list, one case with the phase that deletes it, the shape `internal/sema/corpus_test.go` already uses.
A deferred case that parses clean fails too, so an entry cannot outlive its phase.

Fourteen of the fifteen conformance cases parse clean as generated, and no `conflict` or `prec` annotation needed revising: phase 4 had already settled them by generating a parser.
The fifteenth is `constraints`, which is regex literals and therefore phase 6, and it is the whole of the deferred list.
`externals` means `parser.c` calls a scanner whether or not one produces anything, so `tree-sitter/src/scanner.c` arrives here as a stub that scans nothing.
It is hand-written and lives beside the generated files, which is where every tree-sitter grammar puts it.

The devShell is `mkShell` rather than `mkShellNoCC`, because `tree-sitter parse` builds the parser with a C compiler even though Go needs none.

Reading the trees rather than the exit codes turned up one disagreement the corpus check cannot see, since it is a wrong tree and not an ERROR node: a bare `key` in a class body reads as a modifier on the field after it.
That is `docs/grammar.ebnf` saying less than the prose beside it does, rather than anything the derivation got wrong, so it is in [backlog.md](../backlog.md) as its own change to the canonical grammar.

## Phase 6: the external scanner

`scanner.c` for `regex_lit`, which the built-in lexer cannot separate from unit division without knowing what the parser wants.
Phase 5 left the file as a stub, so this phase writes the scan rather than the plumbing.

The reference implementation solves this with `RescanRegexAt`, and the shape of the problem carries over: the scanner produces a regex only where the grammar allows one.

Done when the constraint cases parse, `matches(/^[^@]+@[^@]+$/)` and `unit N = kg * m / s^2` both read correctly, `testdata/invalid/*/source.tdl` each produce an ERROR node, and `tree-sitter/corpus.sh` has an empty deferred list.

Done.
`valid_symbols` is `RescanRegexAt` asked from the other side, so the scan itself is the reference loop read across: a slash, then anything but a slash, a backslash, or a newline, with a backslash taking the character after it.
End of line and end of file are both unterminated, and the scanner returns false rather than reporting, which leaves the slash to the built-in lexer and makes the file an ERROR instead of a literal running to the end of it.

The two readings coexist in one file: `unit N = kg * m / s^2` beside `matches(/^[^@]+@[^@]+$/)` gives a `unit_expr` and a `regex_lit` and no ERROR.
`corpus.sh` lost the deferred list rather than emptying it, and gained the invalid half, which checks for an ERROR node and not for the message.
`error.golden` is the reference implementation's wording, and a second parser agreeing on the diagnosis is a different promise from agreeing that the file is bad.

`testdata/invalid/unterminated_regex` is new, since an unterminated regex is the failure the scanner introduces and no case covered it.
Both parsers reject it.

One thing is deferred rather than solved.
tree-sitter consults an external scanner during error recovery with every symbol marked valid, so a stray slash in an already-broken file can be read as the start of a regex.
The usual answer is a second, unused external token as a sentinel, which the annotations have no way to express; nothing in either corpus shows the behaviour, so it waits for a case that does.

## Phase 7: wiring

`command make treesitter`, a CI job running it followed by `git diff --exit-code`, and a first `queries/highlights.scm`.

`AGENTS.md` gains the derivation, and `docs/backlog.md` loses the tree-sitter entry it had before this plan existed.

Done when a production added to `docs/grammar.ebnf` without regenerating fails CI.

Done.
The job runs through `nix develop` rather than `tree-sitter/setup-action`, since the regeneration diff is only stable against a pinned CLI and `flake.lock` is what pins it.
`parser.c` carries no timestamp, so the output is deterministic given that version.

Both halves of the done-condition were proved rather than assumed: a production added to `docs/grammar.ebnf` leaves a diff after `make treesitter`, and a spelling deleted from `highlights.scm` fails `go test ./internal/treesitter`.

The query is hand-written, as [treesitter.md](treesitter.md) says it must be, and two checks hold it to the language anyway.
`tree-sitter query` compiles it against the corpus, so a node name the grammar no longer has is an error rather than something quietly left uncolored.
`TestHighlightsCoverKeywords` checks every spelling in `lex.Keywords()` appears in it, because a keyword is an anonymous token and no tree carries its name, which is the check `internal/ebnf` already makes in the other direction.

Neither is a judgment about what should be colored, which is the line the design document draws and this does not cross.

Two of the phase's items were already true.
`AGENTS.md` gained the derivation in phases 5 and 6, and `docs/backlog.md` has no standalone tree-sitter entry left: its "Editor support" section points at this plan instead, which is what losing the entry meant.

The plan named `generate-treesitter`; the target is `treesitter`, matched with `test-treesitter`, and this text is corrected rather than the Makefile.

## Not in this plan

- Editor packaging. Vim, VS Code, JetBrains, and Emacs stay in the backlog, and each wants the language server more than it wants this.
- The language server. Separate work with its own dependencies, and nothing here blocks it.
- Publishing to npm or to a grammar registry, which is a distribution question and not a correctness one.
