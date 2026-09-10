# Corpora, prelude, and examples

The corpora are plain text on purpose, so an implementation in another language can run the same checks.
Adding a directory is enough to add a case; the tests walk them.

## What each corpus promises

`testdata/conformance/*/source.tdl` must parse cleanly and lower to the tree in the sibling `ir.golden`.

`testdata/invalid/*/source.tdl` must fail **to parse**, with an error containing the text in the sibling `error.golden`.
This is the common mistake: a lowering or resolution error does not belong here, because the file parses and the case would not fail.
Those belong in a Go test in the package that raises the diagnostic.
The tree-sitter grammar is held to this corpus too, and must produce an ERROR node for each case.

A case directory holding a `pending` file is skipped, with the file's text as the skip reason.
The corpus is the written-down target, not a record of what already works, so a case ahead of the implementation is correct and the marker is how it says so.

## Canonical form

Every `.tdl` file under `testdata/conformance/` and `prelude/` is stored in canonical form: `tdl fmt <file>` must print it back byte for byte.
`tdl fmt` must also be idempotent, so formatting canonical output is a no-op.

`examples/*.tdl` are deliberately **not** canonical, because they carry `//` comments that `tdl fmt` deletes.
Never suggest formatting them, and never suggest running `tdl fmt` over the directory.

## The prelude

`prelude/std.tdl` is written in TDL and embedded with `go:embed`.
`internal/sema` loads it into an outer scope beneath every file and merges its declarations untagged.

Lowering knows the sugar's spellings (`List`, `Set`, `Map`, `Option`, `Nullable`) and nothing about what they mean.
That is what makes the prelude replaceable, and a change that teaches the compiler what one of them does is the wrong shape.
