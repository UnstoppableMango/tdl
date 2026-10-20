# Implementing the C# source generator

An implementation plan for [csharp-source-generator.md](csharp-source-generator.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan adds `tdl request`, makes the C# backend a library that runs in a compiler, builds `Tdl.Generator` on it, and publishes it to nuget.org.
`tdl-gen-csharp`'s output stays byte for byte the same throughout; the recorded requests in `testdata/csharp/` hold it to that.

F# and a long-lived `tdl` process are out of scope.

## Phase 1: `tdl request`

- `internal/cli` gains `request`, taking `--target` and `--overlay`, and writing framed `plugin.Request` messages followed by the diagnostics.
- An overlay `sema.Loader` serves the texts read from standard input, falling back to `FSLoader`; it shares what it can with `internal/lsp`'s overlay.
- Requests are built with `analysis.Request`, and allowed loss codes are dropped as `gen` drops them.

Done when a Go test runs `tdl request --target csharp` over the smoke model and gets requests equal to those `tdl gen` sends, an overlay text shadows the file on disk, and a lowering error is reported with its range and yields no request.

## Phase 2: the backend as a library

- `Tdl.Gen.CSharp` splits into a library and the `tdl-gen-csharp` executable.
- The library, `Tdl.Plugin`, and `Tdl.DotNet` target netstandard2.0 and net10.0, with a polyfill package and rewrites for what netstandard2.0 lacks.
- Both reference Microsoft.CodeAnalysis.CSharp 4.8; `deps.json` is regenerated.

Done when `make dotnet-test` and `go test ./internal/gen` pass with the recorded output unchanged, and `checks.dotnet-sdk` builds.

## Phase 3: the generator

- `dotnet/Tdl.Generator`, an `IIncrementalGenerator` over `.tdl` additional files, calling `tdl request` once per change of the set and the library in process.
- `TdlPath` and the `Tdl` item in `buildTransitive` props, read through `CompilerVisibleProperty`.
- Diagnostics `TDL0001` to `TDL0004`, the protocol version check, and cancellation.
- `dotnet/Tdl.Generator.Tests` drives it with `CSharpGeneratorDriver`, holding it to the plugin's bytes, and checks step tracking.

Done when the conformance corpus and the smoke model generate through the driver byte for byte as through the plugin, a C# edit reruns no `tdl` call, and each diagnostic is asserted.
A sample project referencing the generator by project reference shows a TDL type in C#.

## Phase 4: navigation and duplicates

- The generator host writes `#line` directives for declarations, fields, and variants; the equality test strips them.
- `TDL0005` reports a declaration generated both here and by checked-in `tdl gen` output.

Done when a compile error placed in generated code is reported at the `.tdl` position, and a test compiling `tdl gen` output beside the generator reports `TDL0005` once per declaration.

## Phase 5: packaging

- `Tdl.Generator` packs as an analyzer with its dependencies and props.
- `release-please-config.json` versions it; `release-please.yml` gains the publishing job with `id-token: write` and `NuGet/login`, gated on the release job's output, added to `unmango/actions` if missing.
- The nuget.org trusted publishing policy names `release-please.yml`; the owner creates it before the first release.

Done when a project restoring the packed `.nupkg` from a local feed generates the smoke model, and a release publishes the package without a stored API key.
