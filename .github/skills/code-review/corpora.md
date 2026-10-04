# Corpora, prelude, and examples

The corpora are plain text so an implementation in another language can run them.
Adding a directory adds a case.

## What each corpus promises

`testdata/conformance/*/source.tdl` parses cleanly and lowers to the sibling `ir.golden`.

`testdata/invalid/*/source.tdl` fails **to parse**, with an error containing the sibling `error.golden`.
A lowering error does not belong here, since the file parses; it belongs in a Go test in the package that raises it.
The tree-sitter grammar must also produce an ERROR node for each case.

A case holding a `pending` file is skipped, with the file's text as the reason.
A case ahead of the implementation is correct.

## Canonical form

Every `.tdl` file in `testdata/conformance/`, `testdata/gen/`, `prelude/`, and `examples/` is canonical: `tdl fmt <file>` prints it back byte for byte.
`TestCorpusIsCanonical` enforces it, and `tdl fmt` is idempotent.

## The prelude

`prelude/std.tdl` is embedded and loaded beneath every file.
Lowering knows the sugar's spellings (`List`, `Set`, `Map`, `Option`, `Nullable`) but not their meaning, which keeps the prelude replaceable.
A change that teaches the compiler what one of them means is the wrong shape.
