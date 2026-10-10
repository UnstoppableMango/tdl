# Implementing transforms

An implementation plan for [transforms.md](transforms.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the transform mode of the plugin protocol, the `transform` declaration, the pipeline between lowering and generation, scoping to targets, and transforms in the language server.
Backends gain nothing: they receive the model the last transform returned, in the shape lowering produces.

Fixes, parameters in a transform header, and `tdl.toml` are deferred in the design and out of scope.

## Testing

Two test transforms live beside the registry and are not compiled into `tdl`, as `internal/gen/echo` is not:

- `internal/transform/lintkey` reports every entity with no `key` directive in its block, and reads `allow(key)` to skip one.
- `internal/transform/audit` adds `created_at` and `updated_at` to every declaration satisfying a class its block names.

Between them they cover a lint with no model and a rewrite that lowering must accept.
The pipeline's tests use `sema.MapLoader`, so no test touches the filesystem.

## Phase 1: protocol

- `plugin.proto` gains `Mode.MODE_TRANSFORM`, `Features.transform`, `TransformRequest`, and `TransformResponse`, each on a number never used; `make generate`.
- `plugin.Describer`, `plugin.Transformer`, and `Description.Transform`; `Serve` and `ServeConn` take a `Describer`, and `ServeConn` gains the transform branch.
- `Transform` on `internal/gen.Subprocess`, resolving `tdl-transform-<name>` on `PATH`.
- `internal/transform` holds the registry, which ships empty; tests register the test transforms themselves.
- `cmd/tdl-transform-lintkey` serves the lint, for the hosts test only.
- `TestHostsAgree` gains a transform column, and `go test ./internal/gen -record` records `testdata/plugin/shop.transform.request.txtpb` and its response for other implementations to replay.

Done when `lintkey` returns the same bytes in process and over a pipe, a generator handshaking under `MODE_TRANSFORM` refuses with a message naming the mode, and `buf breaking` passes.

## Phase 2: unlowering with positions

`internal/unlower` gains an option carrying each `ir.Position` onto the `ast` node it writes, so a tree lowered without printing keeps the model's positions.
A node whose model position is empty keeps the position `unlower` gives it today.

Done when `TestCorpusRoundTrips` also lowers each case's unlowered tree directly, with the option on, to a model equal to the original with positions included.

## Phase 3: syntax and lowering

- `TransformDecl` in `docs/grammar.ebnf` and `docs/spec.md`, with the body of `TargetDecl`.
- `transform` joins `lex.Keywords` and `reserved_word`.
- The parser and `ast` gain the declaration, and `ast.Fprint` prints it.
- `make treesitter` and `make textmate` regenerate the derived grammars, and `tree-sitter/queries/highlights.scm` colors the keyword.
- `ir.TargetBlock.transform` on an unused number; `make generate`; `ir.Dump` prints it.
- `sema` lowers a transform block as it lowers a target block, and reports a transform and a target sharing a name.
- `internal/unlower` writes a transform block back as one.

Done when a conformance case `transform_block` parses, formats canonically, and lowers to its golden with directives tagged on their nodes, an invalid case for a transform and target sharing a name fails with its error, and `TestCorpusRoundTrips` passes over the new case.

## Phase 4: the pipeline

`transform.Run` takes a lowered model and its loader, and for each unscoped block in source order:

1. resolves the block's name to a built-in or a plugin, and checks the block against the declared directives, as `internal/gen` checks a target block;
1. sends a `TransformRequest` and collects the diagnostics;
1. stops on an error diagnostic;
1. with a model in the response, checks that the prelude and every `ir.Extern` are unchanged, gives each positionless own node the block's position, unlowers with positions, and lowers again, reporting lowering's diagnostics under the transform's name.

`tdl check`, `tdl gen`, and `tdl ir` call it after lowering; `tdl ir --transform=false` skips it.
`tdl check` sets `TransformRequest.check`.

Done when `lintkey` fails `tdl check` on a model with an unkeyed entity and passes it once the block says `Order => allow(key)`, `tdl gen` over a model with an `audit` block generates the added fields, an `audit` that returns a field naming an undeclared type fails with lowering's error at the transform block, and a transform editing a prelude declaration fails naming the transform.

## Phase 5: scoped to targets

`internal/gen` reads the `transform` directive beside `out`, declares it repeatable, and reports a name with no transform block.
`tdl gen` runs a target's scoped transforms on the unscoped result, once per target, so two targets naming different transforms each see only their own.

Done when a model with an `audit` block named by its `go` target and not by its `protobuf` target generates the audit fields in Go only, and `tdl check` runs neither scoped transform.

## Phase 6: language server

`internal/lsp` runs unscoped transforms on `textDocument/didSave` and publishes their diagnostics with lowering's.
A transform declaring `reuse` is kept alive for the session and restarted when its binary changes, as `tdl gen --watch` does.
A transform that fails or times out is one diagnostic at its block, never a crashed server.

Done when the VS Code extension test shows `lintkey`'s warning after a save, not after a keystroke, and a killed transform reports at its block and recovers on the next save.

## Phase 7: a transform outside Go

`examples/transforms/` holds a lint written in another language against the generated protobuf, with a README saying how to install it as `tdl-transform-<name>`, and a nix check that runs `tdl check` through it.
It replays the recorded exchange from phase 1.

Done when `nix flake check` runs it, which is what shows a transform needs nothing from Go.
