# Plugins in any language

Design document.
[plugin-sdk-plan.md](plugin-sdk-plan.md) orders the work and says which phases are built.

[plugins.md](plugins.md) promises that built-in backends get no richer interface than a plugin, and that when a backend needs something the protocol cannot express, the protocol changes.
The wire keeps that promise, and the libraries do not.
Every Go backend leans on `backend/internal/emit`, about 1,900 lines that resolve types, follow references, number fields, normalize constraints, and print TDL for round trips.
A plugin in another language receives the same model and must rebuild all of it, and drift from the Go backends wherever it gets one rule differently.

This design moves what is a fact about the model into the host, ships it in the request, and leaves each language a small SDK for what depends on the backend's own decisions.
Its first consumer is a C# backend written in C#, which reaches the .NET compiler's own knowledge of the language through Roslyn.

## What moves

`emit` holds three kinds of code:

| Kind | Share of `emit` | Examples | Where it goes |
| --- | --- | --- | --- |
| Facts | about half | type forms, alias and newtype expansion, references, numbering, length bounds, TDL source text | the host, shipped as data |
| Decisions | a little under half | diagnostics and loss codes, the cascade over what a backend skipped, case styles, round-trip muting | an SDK per language |
| Language policy | a few percent | JSDoc and ApexDoc comment blocks | one backend |

A fact is a pure function of the model, the target, and its directives.
Computing it once in the host makes every backend agree by construction, so a rule fixed once is fixed for every language.

A decision needs something only the backend knows, such as which declarations it refused.
The SDK for it is small, and each SDK holds the same rules over the same facts.

## Analysis

A `Request` gains an `Analysis`, beside the model rather than inside it, since the model is lowering's output and the analysis is derived from it for one target.

```proto
message Analysis {
  repeated TypeView types = 1;  // indexed like Model.types
  repeated DeclView decls = 2;  // indexed like Model.decls
}
```

### Types

A `TypeView` is what `emit.Resolve` answers today, for every entry in the type table:

- the form: primitive, list, set, map, option, nullable, named, extern, parameter, or unit;
- the element and key, as type IDs, with aliases already expanded;
- the primitive's name, the declaration, or the extern;
- a refusal, when the type is one no schema backend can write: a reason, the name it is about, and the position.
  Each SDK words the message, since some name the target language.

Forms are matched by the prelude's spellings, as lowering matches them, so a replaced prelude still reaches every plugin the same way.
A backend with generics reads the parameter and unit forms; one without reads the refusal.

### Declarations

A `DeclView` holds, per declaration:

- whether it is the model's own or the prelude's, instead of each plugin comparing filenames;
- the declarations it names through fields, variants, a newtype base, and type arguments, aliases expanded, which turns the cascade into a short loop over IDs;
- for an enum, whether a variant carries fields, and the discriminant the wire convention gives it;
- for a newtype, the type at the end of its chain;
- for a class, whether it is in a `requires` cycle, and for each instance, whether it can become an implementation and why not;
- the words of its name, `userID` as `user`, `ID`, so every SDK cases names identically.

Fields, variants, and constraints gain the same kind of view:

- **Fields and variants.** Each gets its number under the backend's rule (see [Numbering](#numbering)) and its name's words.
- **Constraints.** Each gets its TDL source text, its `length` bounds normalized to an inclusive range, and whether its `matches` pattern is RE2.
  A length that can never hold becomes a host diagnostic, reported once, before any backend runs.
  A pattern RE2 refuses is not one, since a target with an ECMAScript engine reads a lookahead; the view says so, and a backend whose engine is RE2's decides.

### Directives

The request carries only the directives for its target, so `Find` stops filtering.
`DirectiveSpec` gains a placement, block, declaration, field, variant, or extern, so `tdl` reports a misplaced directive at its position, as it already reports a wrong arity.
It also gains an optional list of allowed values, for directives such as `draft` and `openapi` that take one of a few strings.

## Numbering

A backend that numbers members declares its rules in the handshake, as it declares its directives:

```proto
message Numbering {
  NumberRule fields = 1; // a structure's fields, a fielded enum's variants, a variant's fields
  NumberRule values = 2; // a fieldless enum's variants
  string reserve = 3;    // a declaration directive naming numbers its fields may not take
  string inline = 4;     // a field directive numbering its enum's variants in the field's place
}

message NumberRule {
  int64 max = 1;
  repeated tdl.ir.v1.Range reserved = 2;
}
```

Numbers start at 1.
The host applies `number` pins and allocates the lowest free number to each remaining member, writing it on the member's view.
It reports a pin out of range, on a reserved number, or shared with another pin before generating anything.
Protobuf and Thrift declare rules; Protobuf also names `reserved` and `oneof`.
A backend that does not number declares none and receives no numbers.

## Round trips and import

Under `roundtrip`, six backends write the TDL a target cannot express into annotations.
That text comes from `internal/unlower` and `ast`, which only Go can call.
The analysis therefore carries it, each top-level item, declaration, field, and conformance list as source, plus the imports and the target block, so a plugin copies text rather than printing TDL.
`Analysis.source` holds the items in order, each declaration's with its name and conformance list, and the target block without its `roundtrip` directive; each field's `MemberView` holds its source and its deprecation apart, since a target that carries deprecation itself copies the field without it.
The host fills them only under `roundtrip`.

The import direction has the same problem in reverse: `backend/internal/reverse` builds an `ast.File` and lowers it with `sema` inside the plugin.
An `ImportResponse` may carry TDL source instead of a model, and the host parses and lowers it, as `tdl import` already lowers its own printed output again.
The source is lowered with no loader, as `backend/internal/reverse` lowers what it reads, so it may not import another file.
A plugin that returns a model keeps working.

## SDKs

An SDK wraps the generated protobuf types with what remains:

- a session holding the request and its diagnostics;
- positioned warnings and errors, with the loss codes in [reverse.md](reverse.md#loss-codes);
- the cascade, skipping and warning on every declaration naming a refused one;
- name lookup through a `name` directive, and case styles over the shipped words;
- the round-trip helpers that mute a loss warning a carried annotation recovers.

`emit` becomes the Go SDK, reading the analysis instead of computing it.
Built-in backends keep calling it in process, and `TestHostsAgree` still holds them to the same bytes over a pipe.

The .NET SDK, `Tdl.Plugin`, is a library holding the C# types `protoc` generates for both protos, framing over stdio, and the list above.
A loss code table shared by both SDKs is checked against `reverse.md`, as `emit.LossCodes` is today.

## The C# backend as a .NET plugin

`tdl-gen-csharp` becomes a .NET program built on `Tdl.Plugin` and Roslyn.

- **Syntax.** `SyntaxFactory` builds the output, and `SyntaxFacts` decides which names need `@`, so a keyword a new C# version adds is escaped without anyone updating a list.
- **Checking.** Tests compile the output in process with `CSharpCompilation` and report diagnostics as failures, without writing a project or running `dotnet build`.
  The runtime tests, equality, keys, and defaults, load the compiled assembly and call it.
- **Sharing with F#.** The primitive mapping, namespace derivation, `foreign` and `attribute` parsing, and the .NET regex printer move from `backend/internal/dotnet` into a .NET library.
  The planned `fsharp` backend can then be written in F#, using the F# compiler service the way C# uses Roslyn.
- **Output.** It does not change: [csharp-backend.md](csharp-backend.md) still describes what is generated, and the conformance and smoke tests still build it.

### Shipping

`tdl` resolves `tdl-gen-csharp` on `PATH`, as it resolves any plugin, and the registry loses its `csharp` entry.

The Nix package builds the plugin with `buildDotnetModule` from a lock file, and `tdl`'s package puts it beside the Go binaries.
A user generating C# almost always has the .NET SDK already, but the plugin should not depend on it.
Native AOT produces one binary with no runtime, and `Tdl.Plugin` and Google.Protobuf's binary format are written to trim.
Roslyn is not marked trimming-safe, so the first build publishes a self-contained, ReadyToRun single file, and AOT replaces it once a trimmed build passes the conformance corpus.

## Testing a plugin outside Go

`internal/gen` already records protocol exchanges in `testdata/plugin/` for other implementations to replay.
The .NET test project replays them, so the framing and the handshake are checked against the host's own bytes.
They are recorded in the wire encoding as well as in protobuf text, since the .NET library reads no text format.

The conformance and smoke corpora reach the C# plugin through a Go test that runs `tdl-gen-csharp` from `PATH` and builds what it writes, skipped without it, as the `smithy` and `typescript` import tests skip without their tools.
`checks.gen-csharp` always runs it.

## Open questions

1. **The Go C# backend.** #1082 and #1083 implement phases 1 and 2 in Go, green.
   Merging them pins the output with tests the port must match, at the cost of code deleted later.
   Closing them leaves the design doc and the tests' expectations as the reference.
1. **Analysis on demand.** The host could compute the analysis only for a plugin that asks for it in the handshake. Computing it always is simpler, and the cost is small next to lowering.
1. **gRPC.** Nothing here needs it, since a plugin is one process serving one stream. [grpc-transport.md](grpc-transport.md) designs it separately, as a transport a plugin may choose.
