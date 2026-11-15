# Backends

`backend/internal/emit` holds what every generator shares, reading `internal/analysis` through the request: which declarations belong to the model, reading one target's directives, positioned warnings, case helpers, `Numbers` for field numbering, `Cascade` (skip every declaration naming a skipped one), and `Resolve`, which expands a type reference to the prelude shape or declaration it names.
It also holds the rules more than one target reads: the JSON wire convention (`Discriminant`, `Tag`), `Length` and `Pattern` for the `length` and `matches` constraints, `Narrow`, the narrowed type (a literal union or a tuple) a target with literal and tuple types writes for a `oneOf`, a small integer range, or a list's small exact length, `ECMAScript`, which prints a parsed pattern so a JavaScript engine matches what RE2 does, `ECMAScriptSource`, the same as a RegExp's source for JSON Schema's `pattern`, `NeedsUnicode`, whether that source reads differently without the `u` flag, `Backtracks`, which refuses a pattern shape a backtracking engine takes exponential time on, `Surrogates`, which refuses a pattern naming a code point only JavaScript can see, and `PlanInterfaces`, the class plan for a target that writes a class as a nominal interface.
`Resolve` refuses type parameters and arguments, since Go and Haskell, the targets that generate generics, walk types themselves.
`backend/internal/irtest` builds `*ir.Model` values for backend tests, seeded with the prelude and the `Entity` class.

Every backend reports what it cannot generate as a positioned warning rather than emitting something wrong, and skips a declaration that names a skipped one, so its output compiles.

- `backend/golang` (`go`): one file per declaration, or one file for the whole target that a `file` directive in the target block names.
  `types.go` maps IR types to Go and walks types itself, since `Resolve` refuses type parameters.
  `generics.go` maps type parameters and infers `comparable` for map keys; `classes.go` makes a class an interface with one unexported marker method; `validate.go` turns `where` constraints into `Validate`; `json.go` writes the JSON wire convention under a `json` directive; `foreign.go` maps a declaration to another package's type, and an extern to the type its dependency's `go` block generates.
  `decimal`, `uuid`, and `date` map to placeholders unless a `foreign` directive names a type.
  It imports: `reverse.go` reads a package with `go/parser`, taking constraints back from `validate`'s messages and keys from `Key`; `annotate.go` writes the loss warnings and, under `roundtrip`, `//tdl:` comment directives, carrying a declaration whole where reading the unannotated files back misses something.
  `Normalize` is its normal form for the round-trip corpus.
  See `docs/architecture/go-backend.md`.
- `backend/protobuf`: `.proto` files in the directories the package spells, named for its last segment unless a `file` directive says otherwise.
  A fielded enum is a message holding a oneof of per-variant messages; a newtype expands to its base.
  A `oneof` field directive inlines its enum's single-field variants, and an enum used only that way is not emitted.
  `number` pins a number and unpinned members take the lowest free one; `reserved` writes a `reserved` statement, and a pin on a reserved number or name is refused.
  A type tagged `service` becomes a service; its fields apply primitives tagged `rpc`, with arguments in a `stream`-tagged primitive streamed.
  `format.go` lays out every file with protocompile's printer under its `Legacy` preset, which reproduces `buf format`, so formatting the output changes nothing.
  Tests compile every response with `bufbuild/protocompile`, and `TestOutputIsBufFormatted` runs `buf format -d` when `buf` is on `PATH`.
  It imports: `reverse.go` reads files compiled by `compile.go` into an `ast.File` and lowers it, and `annotate.go` writes the loss warnings and, under `roundtrip`, the options `tdl/annotations.proto` declares.
  `Normalize` is its normal form for the round-trip corpus.
  See `docs/architecture/schema-backends.md`.
- `backend/thrift`: one `.thrift` file per model under `namespace *`, in dependency order.
  A fielded enum is a union of per-variant structs; a newtype is a `typedef`; field ids come from `emit.Numbers`.
  It imports one file: `reverse.go` reads thriftgo's AST, which has no positions, so `locate` scans the source for each definition's keyword; `annotate.go` writes the loss warnings and, under `roundtrip`, `tdl.*` annotations, encoding a value so thriftgo's partial unescaping reads it back.
  `Normalize` is its normal form for the round-trip corpus.
  See `docs/architecture/schema-backends.md`.
- `backend/smithy`: one Smithy IDL 2.0 file per model.
  Each list or map a field holds becomes a named shape (`LineItemList`, `StringLongMap`); a shadowed prelude shape is written `smithy.api#`.
  Non-optional fields are `@required`, and an optional element makes a collection `@sparse`.
  `shapes.go` is the shape model both directions share: generating builds shapes and renders them, and import reads them from the CLI's JSON AST.
  Tests run `smithy validate` when it is on `PATH`; `checks.gen-smithy` always does.
  It imports: `reverse.go` runs `smithy ast` from `PATH` and reads the JSON, which has no positions, so `locate` scans the source for each shape's statement; without the CLI, import and its tests skip.
  `annotate.go` writes the loss warnings and, under `roundtrip`, `metadata tdl` and a `tdl#source` trait on each declaration whose shapes the reader does not read back as it.
  `Normalize` is its normal form for the round-trip corpus.
- `backend/graphql`: one schema per model, output types only.
  Primitives GraphQL lacks, including 64-bit and unsigned integers, become custom scalars declared on use.
  It imports one file: `reverse.go` reads gqlparser's schema document, taking the package from the file's name; `annotate.go` writes the loss warnings and, under `roundtrip`, `@tdl` directives, which the schema then defines.
  See `docs/architecture/schema-backends.md`.
- `backend/typescript`: one `.ts` file per model of JSON wire types, with no runtime code but a newtype's constructor under `construct` or `brand`.
  A set is an array, a map a `Record`, unrepresentable primitives a string, a newtype a plain alias, or under `brand` one intersected with a brand keyed by its qualified name, which import reads back.
  A fielded enum is a union discriminated on `kind`, renamed by a `discriminant` directive.
  `construct.go` writes the constructor, a function named for the newtype that casts its base and throws a `RangeError` for each constraint the type cannot state, `matches` only under `patterns`; import reads one back as a `construct` directive.
  An extern is imported with `import type`, through `backend/internal/ts`'s `imports.go`: from the module a `foreign` directive names, or from its dependency's file by a path relative to this one, which both blocks' `out` directives place.
  A narrowed type from `emit.Narrow` is written as a literal union, which warns only that it reads back as `oneOf`, or a tuple, which import reads back as a list with its `length`.
  Tests run `tsc --noEmit --strict` when it is on `PATH`; `checks.gen-typescript` always does.
  It imports one file: `reverse.go` runs the embedded `outline.js` under `node` with the `typescript` package found from `tsc` on `PATH`, and reads the JSON outline it writes; without both, import and its tests skip.
  `annotate.go` writes the loss warnings and, under `roundtrip`, JSDoc `@tdl` tags, carrying a declaration whole where it loses anything.
  `Normalize` is its normal form for the round-trip corpus.
  `mapper.go` is a TypeScript 7.1 content mapper, served by `tdl-gen-typescript content-mapper`: tsc sends it a `.tdl` file and it lowers the file and returns the module this backend writes, importing an extern from its `.tdl` file; `TestMapperUnderTsc` runs tsc when a 7.1 or later is on `PATH`, through `npm/tdl-typescript/` as it ships.
  See `docs/architecture/typescript-content-mapper.md`.
- `backend/zod`: one `<package>.zod.ts` module per model of Zod schemas, each a `const` and a same-named `type` inferred from it, parsing the JSON wire convention `typescript` types.
  A default is `.prefault()`, which parses it through the schema, checked in `defaults.go` against the rules Zod would apply and the constraints, so one it would refuse warns and is not written rather than failing every parse.
  A narrowed type from `emit.Narrow` is a `z.enum`, a `z.literal`, an enum's `.extract`, or a `z.tuple`, inferring what `typescript` writes.
  `checks.go` writes every other standard constraint as a check (`.min()`, `.max()`, the `codePoints` and `size` refines, `unique`): on a newtype's own schema, or on a schema of a constrained field's own, `LineItemQuantity`, named for its struct and field or by a `newtype` directive, and branded with both; a `oneOf` argument another constraint excludes warns and is left out.
  A `matches` is `.regex()` with a module constant printed by `emit.ECMAScript` (`patterns.go`), only under the `patterns` target-block directive; without it, or for a pattern `emit.Backtracks` or `emit.Surrogates` refuses, it warns.
  Tests run `tsc --strict --exactOptionalPropertyTypes` over every response and parse JSON with it under `node`, when `tsc`, `node`, and a `zod` package on `NODE_PATH` are found; the devShell sets `NODE_PATH` from `nix/zod.nix`, and `checks.gen-zod` always runs them on the smoke model.
  See `docs/architecture/zod-backend.md`.
- `backend/jsonschema`: one `.schema.json` document per model, every declaration under `$defs`.
  A `matches` is printed by `emit.ECMAScriptSource` for a validator's `u` mode; under a dialect that does not ask for it (draft-07, OpenAPI 3.0 and 2.0) one reading differently without it warns under `lossy.constraint`, and one RE2, `emit.Surrogates`, or `emit.Backtracks` refuses warns and is not written.
  Tests compile documents with `internal/schematest.Engine`, which reads the printer's ECMAScript-only escapes back into RE2.
  Tests compile every response with `santhosh-tekuri/jsonschema` and validate instances against it; `checks.gen-jsonschema` runs `check-jsonschema`.
  See `docs/architecture/schema-backends.md`.
- `backend/openapi`: one OpenAPI document per model holding schemas and an empty `paths`, YAML by default or JSON under `format("json")`.
  Tests validate every response against the OpenAPI Initiative's schema for its version, vendored in `backend/openapi/testdata/`; `checks.gen-openapi` runs `vacuum lint` on all three versions.
  See `docs/architecture/schema-backends.md`.
- `backend/salesforce`: Salesforce DX source, one file per component.
  See `docs/architecture/salesforce-backend.md`.
- `backend/haskell`: one module per model, `Shop/Billing.hs` for package `shop.billing` unless a `module` directive names it.
  `foreign.go` maps a declaration, a primitive, or an extern to another module's type, and an extern to the module its dependency's `haskell` block generates; `key.go` writes a key function, and a key record for several fields; `derive` adds `deriving anyclass`.
  `generics.go` writes type variables and kind signatures, derives a declaration holding an applied type variable through standalone deriving with the context reduced from its fields, and decides `Functor`, `Foldable`, and `Traversable` across the whole model.
  `classes.go` writes a class as a type class with a `HasField` superclass per field, and an instance per conformance and per separate instance, plus each instance a required class implies; `requires` on a type constrains its key function.
  See `docs/architecture/haskell-backend.md`.
- `dotnet/Tdl.Gen.CSharp` (`csharp`): a .NET plugin on the SDK, not compiled into `tdl`; `tdl` runs `tdl-gen-csharp` from `PATH`, and `nix/cmd.nix` ships it beside the Go binaries.
  Outside Nix it is the `Tdl.Gen.CSharp` .NET tool on nuget.org, whose `PackAsTool` metadata and package README sit beside it.
  It builds each declaration with Roslyn's `SyntaxFactory` and lays it out with `NormalizeWhitespace`, then `Syntax.cs` decides blank lines and wraps a generated `Equals`; `TdlEquality` is a fixed text.
  A `key` directive becomes a `Key()` method, returning a `<Name>Key` record struct for several fields; a default becomes a property initializer, and the property stops being `required`.
  `Foreign.cs` maps a declaration, a primitive, or an extern to a `global::`-qualified type, and an extern to the type its dependency's `csharp` block generates; `attribute` writes an attribute on a declaration, or on every one from the target block.
  Its tests cannot lower TDL, so `internal/gen/csharp_test.go` records requests: one beside each `testdata/csharp/cases/*/*.tdl`, and one per conformance case and the smoke model in `testdata/csharp/corpus/<case>/`, with the files it generates in `want/`.
  `go test ./internal/gen -record` rewrites them, generating through `tdl-gen-csharp` on `PATH`, and without `-record` compares with them when it is there; CI's race run puts the built plugins on `PATH`.
  The tests compile every response with `CSharpCompilation`, warnings as errors, and run the equality, key, and default checks in process; `checks.gen-csharp` builds the smoke model with `dotnet build` and the SDK's analyzers.
  See `docs/architecture/csharp-backend.md`.
- `backend/likec4`: LikeC4 source for an architecture diagram: `tdl.c4`, the specification of element kinds, the same bytes for every model, and one `<package>.c4` holding a package element and one element per structure, enum, newtype, or class; aliases, primitives, and units have none.
  No Go library parses LikeC4, so tests compare text, and also run `likec4 validate --no-layout` when `likec4` is on `PATH`.
  See `docs/architecture/likec4-backend.md`.
- `backend/debug`: describes the model it was given, to exercise the protocol.
- `backend/yolo` (`yolo`): the delegated backend (see `CONTEXT.md`, #1505), served only as `tdl-gen-yolo` and never compiled into `tdl`, so `tdl` links no model client; `nix/cmd.nix` ships it.
  It holds one conversation per request with a language model through a `Provider`, Anthropic's through `anthropic-sdk-go` with `ANTHROPIC_API_KEY`, and the model answers with `write_file` and `report_loss` tool calls.
  The prompt (`prompt.go`) is the model as TDL, the target block's `language` (required), `prompt`, the loss codes, the JSON wire convention, and `docs/spec.md`, which the `docs` package embeds; `model` picks the model, `claude-opus-5-5` by default, and `effort` the effort it spends, `medium` by default.
  A path that is absolute or holds `..` ends the conversation with an error and nothing written; a `report_loss` with an unknown code or name is refused back to the model, and the rest become positioned warnings under their codes.
  Each extern warns `lossy.unsupported`, since the prompt does not show what its dependency generates.
  A missing key is an error before anything is sent, and a conversation is cut off after 50 turns.
  `TDL_YOLO_REPLAY` names a directory whose `conversation.json` a `Replay` answers from instead, refusing a conversation other than the one it recorded; the recorded system prompt holds a marker in place of the spec.
  `testdata/yolo/<case>/` holds a model, its conversation, the files it wants in `want/`, and `diagnostics.golden`; a case with only a `target.tdl` appends it to the smoke model.
  `go test ./backend/yolo -record` re-records every case live, which needs a key; the committed recordings were written by hand.
  `TestHostsAgree` replays `testdata/yolo/order` in both hosts, its `shipped` row is `pluginOnly`, the round-trip `exempt` list holds it beside `debug`, and `checks.gen-yolo` replays `testdata/yolo/smoke` through the packaged binary.
  It declares itself nondeterministic, so `tdl gen` keeps its output while the request is unchanged and `--verify` needs no key.

`cmd/tdl-gen-<name>` serves each backend as a plugin.
`TestHostsAgree` in `internal/gen` holds each to producing the same bytes in process and over a pipe.
A backend that imports implements `plugin.Importer`, declares `Reverse`, and has a `reverse` column in `shipped`, which `TestImportHostsAgree` runs; `TestReverseIsDeclared` keeps the three in step.
A warning a user can silence carries a loss code; `host.LossCodes`, which `emit.LossCodes` re-exports, must match the table in `reverse.md`.
Output loss, what the output drops, is written with `emit.Session.Lossy` and always reported; read-back loss, what only import would miss, with `emit.Session.Lose`, reported under a bare `readback` directive and never under `roundtrip` (see `reverse.md`'s Loss codes).
A backend added to the registry needs a row in the `shipped` table in `internal/gen/hosts_test.go` (`TestEveryBuiltinHasARow`) and, if shipped, an entry in `nix/cmd.nix` (`TestPackagedBackendsShip`); a Go backend served only as a plugin has a `pluginOnly` row.

`docs/architecture/schema-backends.md` maps the seven schema backends.
`docs/architecture/reverse.md` is the import direction, target language to TDL, and #972 tracks the remaining importers.
`backend/internal/roundtrip` runs a backend forward and back over `testdata/roundtrip/<target>/<case>/`; its `targets` table holds every backend but those in `exempt` (`debug` and `yolo`), with each one's normal form, and `TestEveryTargetIsCovered` checks it against `cmd/`.
A model-first case's `lossy.golden` holds the codes generating with `readback` reports, and generating with `roundtrip` must warn exactly what generating with neither does.
`TestConformanceComesBack` also takes every conformance case without an import through each target that imports, and `TestLossKinds` holds each such target to both kinds of loss.
`testdata/gen/smoke/source.tdl` exercises the whole mapping, with a target block for each schema backend and for `salesforce`; the nix checks generate from it and run each language's tool on the output.

## Feature tables

`docs/backends/<target>.md` is a backend's page for model authors, holding a table of what it does with each feature of the language between `<!-- tdl:features <target> -->` and `<!-- /tdl:features -->`.
`internal/features` measures it and `tools/features` (`make features`) writes it; `TestTablesAreCurrent` compares and never writes.
A row is a directory of `testdata/features` holding `source.tdl`; a subdirectory named for a backend holds its own source, for a feature that backend reaches only through its own directives.
A source's target block is written for `feature` and renamed for each backend, in its imports too.
Generate is No on an error or a warning without a loss code, Warns on the loss codes the row owns (`Row.Codes`), and Yes otherwise, since a source cannot use a feature alone.
Generate leaves read-back loss out, as `tdl gen` does.
Import generates with `readback` and without `roundtrip` and reads the files back, grading only read-back loss: the warnings `readback` adds and those import reports, never the output loss Generate shows.
The same model is Yes, as is a difference output loss alone explains; a difference with no loss code anywhere is No.
`features.toml` beside a backend's source (`backend/<package>/`, or `dotnet/Tdl.Gen.CSharp/` for `csharp`) holds the cells a measurement cannot: `spelling`, `notes`, and `planned`, which shows in place of Generate and fails once the row generates.
A cell may not hold a `|`.
An Import cell a tool missing from PATH, or `-short`, leaves unmeasured keeps its committed text, so `make features` needs `smithy` and `tsc` to change those cells.
A plugin this repository serves, `csharp`, is measured through `tdl-gen-csharp` when it is on PATH, as in CI's race run, and otherwise keeps every committed cell.
Every target but `debug` is embedded exactly once.
Two matrices set one column of every table side by side, each cell its first word: `generate` in `README.md` and `import` in `docs/backends/README.md`, where a backend that imports nothing has no column.
Each is embedded exactly once, and reads the tables after committed cells fill unmeasured ones.
A table naming no target or matrix belongs to a backend with no code: it is written by hand, holds every row in order with only `Planned (phase N)` or `No` in Generate and Import, and `CheckPlanned` checks it without rewriting it.
When the backend is built and registered, the same table is measured, and its `features.toml` takes the phases.
The tables stay out of `docs/architecture/`, which is for building the backend, not using it.
