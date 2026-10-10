# Reverse backends

Design document.
[reverse-plan.md](reverse-plan.md) orders the work.

A backend reads a model and writes a target language.
A reverse backend reads the target language and writes a model, so `tdl import` turns a `.proto`, `.thrift`, `.smithy`, `.graphql`, `.ts`, `.schema.json`, Go package, or Salesforce DX source into TDL.

## Goal

Every backend except `debug` round-trips, in both directions:

- **Model first.** `tdl` to target to `tdl` lowers to an IR equal to the original, positions ignored.
- **Schema first.** A hand-written schema imports to TDL and regenerates to a schema equal to the original under the backend's normalizer (see [Equality](#equality)).

A conversion that cannot round-trip is allowed and warns with a loss code.
A loss code can be silenced in `tdl.toml`.

## Shape

A reverse backend returns an `ir.Model` and diagnostics.
It never writes TDL text.

The host turns the model into TDL: `internal/unlower` builds an `*ast.File` from the model's own declarations and its target blocks, and `ast.Fprint` prints it.
The result is lowered once more before it is written, so `tdl import` never writes a file `tdl check` rejects.

This keeps one printer for every backend, and a plugin in another language already speaks IR.

Unlowering rebuilds sugar from `Type.wrote`, so `[T]`, `T?`, and `{K -> V}` print as written.
It writes only the declarations `emit.IsOwn` reports, so the prelude is never printed back.

## Protocol

[plugins.md](plugins.md) is extended rather than forked, and every change is additive.

- `Handshake.mode` is `MODE_UNSPECIFIED`, the zero value, which generates, or `MODE_IMPORT`.
- `Features.reverse` declares that a plugin answers `IMPORT`.
  The host refuses import mode when the reply lacks it, so a plugin that ignores `mode` is never sent an `ImportRequest`.
- `ImportRequest` carries the target name, the source files, and the loss codes the user allowed.
- `ImportResponse` carries the model and diagnostics.
- `Diagnostic.code` is a stable string, set on every loss warning.

In Go, `plugin.Importer` is an optional interface beside `plugin.Backend`.
`internal/gen.Subprocess` implements it too, so a built-in backend gets nothing a plugin cannot.

## Loss codes

`backend/internal/emit/loss.go` holds the one list, so the forward and reverse directions name a loss the same way.

| Code | What is lost |
| --- | --- |
| `lossy.primitive` | a primitive's width or kind, such as `int` and `int64` both written `int64`, or `uuid` written `string` |
| `lossy.newtype` | a newtype expanded to its base |
| `lossy.collection` | `Set` written as a list |
| `lossy.optional` | `Option` against `Nullable`, or optionality on a message-typed field |
| `lossy.struct-kind` | entity, value, or mixin |
| `lossy.include` | a mixin `include`, flattened into the including struct |
| `lossy.alias` | an alias expanded at each use |
| `lossy.constraint` | a `where` constraint |
| `lossy.owned` | the `owned` modifier |
| `lossy.default` | a field default |
| `lossy.key` | a `key` directive |
| `lossy.name` | a `name` directive against the naming convention |
| `lossy.number` | a pinned number against an assigned one |
| `lossy.order` | declaration order |
| `lossy.doc` | doc comment text or indentation |
| `lossy.generic` | type parameters |
| `lossy.class` | a class |
| `lossy.unit` | a unit |
| `lossy.unsupported` | a target construct TDL cannot express |

`Session.Lossy(code, position, format, args...)` writes the warning.
The host drops a warning whose code is allowed, so a backend may ignore the allowed list.

Generating, a backend warns once per lost fact.
Importing, a backend warns when the source leaves a fact ambiguous and when a construct has no TDL form.

## Round-trip annotations

A bare `roundtrip` directive in a target block makes a backend write every fact its mapping would lose, in the target's own extension syntax.
Without it, output is unchanged and each lost fact is a loss warning.

| Backend | Annotation |
| --- | --- |
| `protobuf` | custom options declared in `tdl/annotations.proto`, such as `[(tdl.field) = {source: "id: uuid"}]` and `option (tdl.message) = {kind: KIND_ENTITY};` |
| `thrift` | annotations with `tdl.` keys, such as `(tdl.source = "id: uuid")` and `(tdl.kind = "entity")` |
| `smithy` | traits in the `tdl` namespace, defined in a generated `tdl.smithy` |
| `graphql` | an `@tdl` directive, defined in the schema it is used in |
| `typescript` | JSDoc `@tdl` tags |
| `jsonschema` | `x-tdl` keywords, which validators ignore |
| `go` | `//tdl:` comment directives |
| `salesforce` | a `tdl:` tail in an XML `description`, and `@tdl` ApexDoc tags |

The Go reverse also reads what the output already says: a constraint from the `fmt.Errorf` text in `Validate`, which writes it as TDL, and a `key` from `Key()`.

A declaration the target cannot express at all, which generating skips with a warning, is carried whole as TDL, so a `roundtrip` output warns about nothing it carries.

A schema written by hand carries no annotations and needs none.
It imports to TDL plus a target block holding what regeneration reads: `number`, `name`, `option`, `file`, `package`, `edition`, `discriminant`, and the like.
A reverse backend writes only directives its forward twin declares.

## Configuration

`tdl.toml` is found by walking up from the input file; the nearest one is read, and one further up is not merged in.
`tdl import` walks up from its first file.
This design reads one table of it:

```toml
[lossy]
allow = ["lossy.order"]

[lossy.protobuf]
allow = ["lossy.collection"]
```

`[lossy]` applies to every backend, and `[lossy.<target>]` to one, in both directions.
`--allow-lossy` on `tdl gen` and `tdl import` adds codes to the list.

Silencing is diagnostics policy, not type mapping, which is why it lives in the manifest and not in a target block ([workflow.md](workflow.md)).

## Command

```text
tdl import --from <target> [-o out.tdl] [--package <path>] [--allow-lossy <code>,...] <files>...
```

`--from` resolves through `gen.Resolve`, so a plugin on `PATH` imports the same way a built-in does.

## Readers

| Target | Reader |
| --- | --- |
| `protobuf` | `bufbuild/protocompile` |
| `thrift` | `cloudwego/thriftgo` |
| `graphql` | `vektah/gqlparser/v2` |
| `go` | `go/parser` |
| `typescript` | the TypeScript compiler API, run by an embedded script under `node` |
| `smithy` | `smithy ast`, the CLI's JSON AST |
| `jsonschema` | `encoding/json`, after `santhosh-tekuri/jsonschema` compiles the document |
| `salesforce` | `encoding/xml` and a recognizer for Apex |

`typescript` and `smithy` need `node` and the Smithy CLI on `PATH`, and say so when either is missing.
Each is replaced by a Go parser: Smithy IDL is small enough to parse from its published grammar, and TypeScript moves to `microsoft/typescript-go` once its AST is importable.

## Equality

Round-trip tests compare models with positions ignored.

Schema-first tests compare each schema in a normal form:

| Target | Normal form |
| --- | --- |
| `protobuf` | `FileDescriptorProto` without source info |
| `thrift` | the thriftgo AST as JSON without comments, each kind of definition sorted by name |
| `graphql` | the `gqlparser` schema document, formatted without comments, its definitions sorted by name and its scalars without descriptions |
| `go` | declarations through `go/format`, keeping doc comments only |
| `typescript` | the compiler API's JSON |
| `smithy` | the `smithy ast` JSON |
| `jsonschema` | the parsed document with object keys sorted |
| `salesforce` | the parsed XML, and the Apex recognizer's output |

Formatting and ordinary comments are not part of a model, so they are not compared.

## What is not here

A construct TDL has no form for is `lossy.unsupported`, and the declaration reaching it is skipped the way `emit.Cascade` skips one generating.
Some of these want TDL surface, such as a GraphQL interface as a class or a Smithy operation as a service; each goes to [backlog.md](../backlog.md) rather than into this design.

The `debug` backend has no reverse.
