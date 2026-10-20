# Implementing foreign imports

An implementation plan for [foreign-imports.md](foreign-imports.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan lets an `import` name a target-language file, read through that target's reverse backend at lowering time.
It builds extension dispatch, the `from` clause, pointing the source's own target back at the original, vendoring into other targets, the language server, watch mode, and the flake module's `tools` option.

Following a foreign file's own imports, directories as paths, and an on-disk cache are out of scope.
When foreign imports are followed, they resolve through each ecosystem's own tooling; see [backlog.md](../backlog.md#foreign-imports-that-import).

## Testing

Lowering's tests use `sema.MapLoader` with an in-process registry, so they run no subprocess and touch no filesystem.
`internal/gen/echo` claims the extension `.echo` for the protocol tests, so the handshake and dispatch are tested without a real reader.
Tests for a reader that needs a tool skip without it, as the readers' own tests do, and a nix check runs them all.

## Phase 1: loader and dispatch

- `sema.Loader` returns either TDL source or a foreign file with its target; `FSLoader` decides by extension, from a fixed table of the built-in readers.
- `sema` sends an `ImportRequest` through a registry it is given, unlowers a returned model, and lowers it as the imported package.
- Reader diagnostics are reported at the `import` line with the reader's position as related information, and loss codes are dropped under `[lossy]` and `--allow-lossy`.
- A missing tool is an error at the `import` line naming it.
- Only exported declarations are nameable; an unresolved name the foreign file imports is a diagnostic at the `import` line.
- `.go` and `.proto` only.

Done when a conformance-style test under `testdata/foreign/` lowers a model importing a `.go` and a `.proto` file through `tdl check`, a lossy reader warning is silenced by `tdl.toml`, and an unknown extension fails naming it.

## Phase 2: `from` and plugin extensions

- `plugin.proto` gains `Features.extensions` on a number never used; `make generate`; `plugin.Description.Extensions`.
- The handshake refuses extensions without `reverse`.
- Every built-in reader declares its extensions, replacing phase 1's table; the registry reports an extension claimed twice.
- `ImportSpec` in `docs/grammar.ebnf` and `docs/spec.md` gains the optional `from` target; the parser, `ast`, and `ast.Fprint` follow; `from` stays contextual.
- `make treesitter` and `make textmate` regenerate the derived grammars.
- `.thrift`, `.graphql`, `.smithy`, and `.ts` are read.

Done when `echo` claims `.echo` over a pipe and is dispatched to by extension, a second claimant makes the extension require `from`, a conformance case with `from` parses, formats canonically, and lowers, and `buf breaking` passes.

## Phase 3: the source's own target

- `ImportResponse` gains `placement`, target block directives on a number never used, beside the existing model and `source`; `make generate`.
- The Go reader returns `package` from the nearest `go.mod`; the TypeScript reader returns `out`; the other readers return what their forward twin reads for a dependency.
- `sema` lowers `placement` as the imported package's block, whether the package came as a model or as TDL `source`, for the reader's target, so each backend's existing extern handling refers to the original.
- A directive on a foreign declaration for its source target is an error.

Done when a model importing `types.go` generates Go that compiles against the original package with no copy of its types, and the same for a `.ts` import generating TypeScript under `tsc --strict`.

## Phase 4: vendoring

- `tdl gen` generates a foreign package's declarations as their own unit for every target but the source's, under the importer's `out` at a path named for the source's package.
- `lossy.vendored` joins `emit.LossCodes` and the table in `reverse.md`, warned once per foreign import and target.
- Target block directives on a vendored declaration reach its `ir.Extern` and the vendored declaration.

Done when a model importing `types.go` generates TypeScript and protobuf that pass `tsc --strict` and protocompile with the vendored units, `lossy.vendored` is silenced by `tdl.toml`, and a `name` directive on a vendored declaration renames it in both the unit and its uses.

## Phase 5: language server

- `internal/lsp` caches each foreign package by content hash and target, in memory, and serves unsaved foreign text through the overlay loader.
- Hover and go to definition on a foreign name jump into the foreign file.

Done when editing `types.go` in the editor updates diagnostics in the importing `.tdl` file without a save, an unchanged foreign file runs its reader once per session, and go to definition on `gotypes.Order` opens `types.go` at its declaration.

## Phase 6: watching and nix

- `tdl gen --watch` and `tdl play` watch foreign files.
- The flake module's `tdl-check` and `tdl-gen` checks take `tools`, empty by default.
- `checks.flake-module` imports a `.ts` file with `tools = [ nodejs typescript ]`.

Done when touching a watched `.go` file regenerates, and `nix flake check` builds the flake module check with a foreign import.
