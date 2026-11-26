# Tests and corpora

## Tests and goldens

After any change to lowering or `ir.Dump`, run `go test ./internal/sema -update` and read the diff.
`testdata/plugin/` holds recorded protocol exchanges for other implementations to replay, regenerated with `go test ./internal/gen -record`.
`internal/sema/corpus_test.go` asserts the conformance corpus lowers with no diagnostic.

## Corpora

`testdata/conformance/*/source.tdl` must parse and lower to the sibling `ir.golden`.
`testdata/invalid/*/source.tdl` must fail with an error containing the sibling `error.golden`.
Both are plain text so other implementations can run them, and `parser/conformance_test.go` walks them, so adding a directory adds a case.
A case holding a `pending` file is skipped with its text as the reason; the phase implementing the construct deletes it.

Every `.tdl` file in `testdata/conformance/`, `testdata/gen/`, `prelude/`, and `examples/` is canonical: `tdl fmt <file>` prints it back byte for byte (`TestCorpusIsCanonical`).
`tdl fmt` is idempotent.
At the top level, a blank line between comment groups, or between a comment and the following declaration, survives formatting.
Inside a body, the formatter owns blank lines.
