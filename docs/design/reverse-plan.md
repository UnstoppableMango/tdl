# Implementing reverse backends

An implementation plan for [reverse.md](reverse.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Phase 1: unlower

`internal/unlower` turns an `ir.Model` into an `*ast.File`, and a comparison ignores positions: an `ir.Dump` option that omits them, or protobuf equality after clearing them.

A variant's payload prints expanded only when `Variant.End.Line` is set (`ast/print.go`), so unlowering sets positions or the printer decides from content.

Done when every `testdata/conformance/*/source.tdl` parses, lowers, unlowers, prints, parses, and lowers to an equal model.

## Phase 2: protocol, loss codes, and configuration

- `plugin.proto` gains `Handshake.mode`, `Features.reverse`, `ImportRequest`, `ImportResponse`, and `Diagnostic.code`; `make generate`.
- `plugin.Importer`, the `IMPORT` branch in `ServeConn`, and `Import` on `internal/gen.Subprocess`.
- `emit.Lossy` and the code list.
- `internal/config` reads the `[lossy]` table of `tdl.toml`, and the diagnostic filter both commands share applies it.
- `tdl import`.
- `TestImportHostsAgree`, beside `TestHostsAgree`, reads a reverse column in the `shipped` table.

Done when a test backend imports the same model in process and over a pipe, and an allowed code prints nothing.

## Phase 3: round-trip harness

`backend/internal/roundtrip` runs a backend forward and back.
Its corpus is `testdata/roundtrip/<target>/<case>/`:

- a `source.tdl` is generated with `roundtrip` on, and must come back equal, and with it off, and must warn exactly the codes in `lossy.golden`;
- target files with an `expected.tdl` must import to it and regenerate equal under the target's normal form.

`testdata/gen/smoke/source.tdl` is a model-first case for every target.

Done when the harness runs an empty corpus for every target.

## Phase 4: one target at a time

In order: `protobuf`, `thrift`, `graphql`, `go`, `typescript`, `smithy`.
Each is:

1. loss warnings generating;
1. `roundtrip` annotations;
1. the reader;
1. import served by `cmd/tdl-gen-<name>`;
1. corpus cases in both directions;
1. a section in the target's design document.

Done, for a target, when its corpus passes and `testdata/gen/smoke` round-trips with no warning.

## Phase 5: Salesforce

Tracked on its own.
XML reads into the `customObject` and `customField` structs the backend writes with, and a recognizer reads Apex, first the shapes `apex.go` writes and then hand-written classes.

Done when the same bar as phase 4 holds.

## Phase 6: Go parsers

Smithy IDL is parsed in Go from its published grammar, and TypeScript moves to `microsoft/typescript-go` once its AST is importable.
The corpus is unchanged, which is what shows the swap changed nothing.

Done when `tdl import` needs nothing on `PATH`.
