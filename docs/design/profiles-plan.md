# Implementing profiles

An implementation plan for [profiles.md](profiles.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds profile declarations, kind selectors, `with` on a target block, and the embedded `std` profiles.
Backends gain nothing: a profile reaches them as resolved directives, and the plugin protocol in [plugins.md](plugins.md) changes only by the two IR fields profiles.md names.

## Testing

The conformance corpus carries the syntax and the lowering: each case is a `source.tdl` and an `ir.golden`, and each invalid case an `error.golden`.
`TestCorpusIsCanonical` holds every new `.tdl` file to `tdl fmt`.
The derived grammars are checked by `go test ./internal/treesitter` and `go test ./internal/textmate`, and `make test-treesitter` parses the corpus.

## Phase 1: syntax

`ProfileDecl`, `with` in `TargetDecl`, and the kind selector in `Path`, in `docs/grammar.ebnf` and `docs/spec.md`.
`profile` joins `lex.Keywords` and `reserved_word`; `@` joins the punctuation.
The parser and `ast` gain the declaration and the selector, and `ast.Fprint` prints both.
`make treesitter` and `make textmate` regenerate the derived grammars, and `tree-sitter/queries/highlights.scm` colors the keyword.

Done when conformance cases `profiles` and `kind_selectors` parse and format canonically, and invalid cases for a field path in a profile and for `with` naming two profiles fail with their error.

## Phase 2: lowering

`sema` resolves the name after `with`, follows the chain, and reports a cycle and a target mismatch.
Entries resolve layer by layer with the precedence in [profiles.md](profiles.md#precedence), and kind selectors take their place on the ladder.
`proto/tdl/ir/v1/ir.proto` gains `Directive.from_profile` and `TargetBlock.profile`; `make generate` rewrites `ir/ir.pb.go`, and `ir.Dump` prints both.
`go test ./internal/sema -update` regenerates the goldens, and the diff touches only the new cases.

Done when the goldens show each profile entry on the nodes it selects, with `from_profile` set, and a block entry beating a profile entry of higher specificity.

## Phase 3: shipped profiles

`prelude/std/` holds one embedded file per target, loaded by `sema` when a block names a profile under `std.<target>`.
A test lowers every embedded file.
`tdl ir` on a file applying a `std` profile needs no import.

Done when `with std.<target>.<name>` resolves from a file with no imports, and a broken shipped profile fails `go test ./prelude`.

## Phase 4: editors

`refs.go` records the name after `with` and a kind selector, so the language server goes to a profile's definition, including one embedded under `std`, the way it serves prelude names.
Hover prints the profile with `ast.PrintDecl`.

Done when go to definition and hover work on a `with` naming a shipped profile in VS Code.
