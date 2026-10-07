# Round-trip corpus

Cases for the [reverse backends](../../docs/design/reverse.md), run by `go test ./backend/internal/roundtrip`.

Each target has a directory named for it, and each case is a directory inside that one.
A case runs one direction:

- **Model first.** `source.tdl` is generated without annotations and must warn exactly the loss codes in `lossy.golden`, one per line, sorted.
  Generated again with a `roundtrip` directive in its target block, it must import back to an equal model, positions aside.
- **Schema first.** Every other file is the target's own source.
  It must import to `expected.tdl`, warning exactly the codes in `lossy.golden`, and `expected.tdl` must regenerate it, equal under the target's normal form.

A missing `lossy.golden` lists no codes.
`-update` rewrites `lossy.golden` and `expected.tdl` from what the backend does; read the diff.

`testdata/gen/smoke/source.tdl` also comes back, with annotations, from every target that imports.
A target that does not import yet may have no cases.
