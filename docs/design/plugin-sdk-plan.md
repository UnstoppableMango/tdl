# Implementing plugins in any language

An implementation plan for [plugin-sdk.md](plugin-sdk.md).
Phases are ordered by dependency, and each states what makes it done.

Phases 1 to 4 are done.

## Scope

This plan moves the facts in `backend/internal/emit` into the host, ships them in the plugin request, builds a .NET SDK, and rewrites the C# backend as a .NET plugin.
Every Go backend's output stays byte for byte the same until the C# port; the corpora and `TestHostsAgree` hold them to it.

The F# backend and gRPC are out of scope.

## Phase 1: types and references in the request

- `plugin.proto` gains `Analysis`, `TypeView`, and `DeclView`, on numbers never used; `make generate`.
- `internal/analysis` computes them from a model and a target, moving `Resolve`, `Expand`, `References`, `Fielded`, `Discriminant`, ownership, and name words out of `emit`.
- `internal/gen` attaches the analysis to every `Request`.
- `emit.Resolve` and the rest read the analysis, keeping their signatures, so no backend changes.
  A session without an analysis, as in a test that builds a request by hand, derives one.

The analysis also carries each field's and variant's words, which phase 3 extends with numbers.

Done when every backend's tests and the round-trip corpus pass unchanged, and `TestHostsAgree` holds over a pipe.

## Phase 2: directives

- The request carries only its target's directives.
- `DirectiveSpec` gains placement and allowed values, and `internal/gen/check.go` checks both, plus repeats on the target block itself.
- Each backend declares the placement of its directives, and drops its own placement checks where the host now makes them.

Done when a `namespace` on a declaration and a `draft("2019")` are reported by `tdl` with their positions, before any backend runs.

## Phase 3: constraints and numbering

- The analysis carries each constraint's source, its normalized `length` range, and its pattern's RE2 validity; `emit.Length`, `Pattern`, and `ConstraintText` read them.
- `HandshakeReply` gains `NumberRule`; the host numbers members, and `emit.Numbers` reads the result.
- Protobuf and Thrift declare their rules.

Done when the protobuf and thrift goldens are unchanged and a pin on a reserved number is reported once by the host.

## Phase 4: round trips and import

- Under `roundtrip`, the analysis carries the source of each item, declaration, field, and conformance list, and the imports and target block; `emit`'s round-trip helpers read it.
- `ImportResponse` gains a `source` field, which the host parses and lowers; `backend/internal/reverse` keeps working for Go backends.

Done when the round-trip corpus passes, and a test importer returning source produces the same model as one returning a model.

## Phase 5: the .NET SDK

- `dotnet/` holds a solution: `Tdl.Plugin`, its tests, and a `Tdl.Plugin.Debug` port of `backend/debug`.
- The C# types are generated from both protos by `buf generate`.
- `Tdl.Plugin` holds framing, the handshake, the session, diagnostics and loss codes, the cascade, name lookup, case styles, and the round-trip helpers.
- Its tests replay `testdata/plugin/` and check the loss codes against `reverse.md`.
- `nix/` builds it with `buildDotnetModule`, and the devShell carries the .NET SDK it already has.

Done when the debug port answers a recorded exchange byte for byte, and `tdl gen` runs it from `PATH`.

## Phase 6: the C# backend as a .NET plugin

- `Tdl.Gen.CSharp` builds output with Roslyn `SyntaxFactory` and escapes names through `SyntaxFacts`.
- It ports phases 1 and 2 of [csharp-backend-plan.md](csharp-backend-plan.md), with the shared .NET pieces in `Tdl.DotNet`.
- Its tests compile output with `CSharpCompilation` and run the equality, key, and default checks in process.
- A Go test runs the conformance and smoke corpora through `tdl-gen-csharp` from `PATH`; `checks.gen-csharp` always does.
- The registry drops `csharp`, the Go `backend/csharp` is deleted if it was merged, and `nix/cmd.nix` ships the plugin beside the Go binaries as a self-contained single file.

Done when the corpora generate the same C# they generated in Go, up to formatting, and build.
Phases 3 to 6 of `csharp-backend-plan.md` then continue in C#.

## Phase 7: native AOT

- Publish `tdl-gen-csharp` with Native AOT, and fix or suppress each trimming warning Roslyn raises.

Done when the AOT binary passes the corpora, or this phase records which Roslyn warning blocks it.
