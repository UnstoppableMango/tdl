# TDL types live in C# editors

Design document.
Phase 1, `tdl request`, is built; the rest is tracked in [#1199](https://github.com/UnstoppableMango/tdl/issues/1199).

`tdl gen` writes C# to disk, and a project compiles what was written.
The C# a model generates is stale from the moment a `.tdl` file changes until someone runs `tdl gen` again, and an editor knows nothing of a declaration added a minute ago.

A Roslyn incremental source generator closes that gap.
The compiler and the editor run it in process, hand it the project's `.tdl` files as `AdditionalFiles`, including unsaved buffers, and compile what it returns beside the project's own code.
A field added to a TDL declaration completes in C# before the file is saved.

The generator is an alternative a project opts into.
`tdl gen`, `tdl gen --verify`, and the Nix flake module stay the hermetic path, which checks generated output in and builds it without a running compiler host.

## Lowering stays in `tdl`

Lowering is Go, in `internal/sema`, and a source generator runs inside the C# compiler on netstandard2.0.
The generator cannot produce a model itself, and porting lowering to C# would fork the language.

So the generator asks `tdl` for one.
A new subcommand returns the `plugin.Request` that `tdl gen` would send the `csharp` backend, and the generator runs the C# backend over it in process:

```text
.tdl buffers ──stdin──▶ tdl request --target csharp ──stdout──▶ Requests
                                                                   │
                         compilation ◀── Tdl.Gen.CSharp (in process)
```

A process per change is enough.
Lowering a model takes milliseconds, and the generator calls `tdl` only when a `.tdl` text changes, never on a C# edit.
A long-lived `tdl` process is an optimization to add if latency shows, and is not designed here.

Running Go lowering compiled to WebAssembly inside the generator was rejected: it would ship a native WebAssembly runtime per platform inside an analyzer package.

## `tdl request`

```text
tdl request --target csharp [--overlay] [file ...]
```

It writes two framed messages, as the plugin protocol frames a message, and exits: a `Handshake` carrying its `framing_version` and `ir_version`, then one `RequestSet`, which `plugin.proto` gains:

```proto
message RequestSet {
  repeated Request requests = 1;       // one per file whose model has a block for the target
  repeated Diagnostic diagnostics = 2; // the protocol's Diagnostic, with its file and range
}
```

One message holds both, so the generator never has to tell a request from a diagnostic by position in the stream, and an empty set is a valid answer.
`tdl request` builds each request through `analysis.Request`, as `internal/gen` does, so the generator receives exactly what `tdl gen` sends, analysis and directive checks included.

With `--overlay`, standard input carries one framed `Overlay`, a list of path and text pairs:

```proto
message Overlay {
  repeated OverlayFile files = 1;
}

message OverlayFile {
  string path = 1;
  string text = 2;
}
```

A sema `Loader` serves an import from the overlay first and from disk otherwise, as `internal/lsp`'s overlay serves an editor's unsaved buffers.
An import outside the project, and a foreign import, are read from disk.

The diagnostics are parse and lowering errors, directive check failures, and the refusals `analysis.Request` returns.
A file with an error yields no request.
Loss-coded warnings that `tdl.toml` or `--allow-lossy` allows are dropped here, so `tdl.toml` means the same to both workflows.

The command writes nothing to disk.

## The generator

`Tdl.Generator` is an `IIncrementalGenerator`.

### Inputs

The package's props declare an MSBuild item, `Tdl`, defaulting to `**/*.tdl` under the project, and add every `Tdl` item to `AdditionalFiles`.
The generator filters `AdditionalTextsProvider` to `.tdl` paths, collects the texts into one value compared by path and content, and makes one `tdl request` call when that value changes.
Imports cross files, so one call over all of them lowers each shared file once.

Every `.tdl` file is lowered; only files whose model has a `csharp` target block generate.
The directives in that block mean what they mean to `tdl gen`, so a model moves between the two workflows without edits.
`out` is ignored, since the generator writes into the compilation.

`TdlPath`, a compiler-visible MSBuild property, names the executable, and defaults to `tdl` on `PATH`.
The devShell and the home-manager module already put it there.
The package bundles no `tdl` binary.

### Output

Each request goes to the same generator `tdl-gen-csharp` runs, in process.
A file's hint name is the path `tdl gen` would write, relative to the `.tdl` file's directory, such as `shop/Order.g.cs`, so two models declaring `Order` in different namespaces do not collide.

The generator adds `#line` directives mapping each declaration, field, and variant to its position in the `.tdl` file.
Go to definition on a generated type lands on the TDL declaration, and a compile error inside generated code points there.
`tdl gen` does not write them, so checked-in output carries no absolute paths.

### Diagnostics

Each diagnostic in the `RequestSet` becomes a Roslyn diagnostic at its file and range, so it shows in the editor's error list and an error fails the build:

| ID | Severity | Meaning |
| --- | --- | --- |
| `TDL0001` | error | `tdl` was not found, or exited without a reply |
| `TDL0002` | error | `tdl` speaks a different protocol version |
| `TDL0003` | error | a parse, lowering, or directive error |
| `TDL0004` | warning | a backend or lowering warning; its loss code follows the message |
| `TDL0005` | warning | a declaration is generated both here and by checked-in `tdl gen` output |

`TDL0001` and `TDL0002` generate nothing.
The version check is the plugin handshake's, run the other way: the generator reads the `Handshake` `tdl request` writes first, compares its `framing_version` and `ir_version` with its own, and names both on a mismatch.

`TDL0005` finds `tdl gen` output in the compilation by the header every generated file starts with, and reports once per declaration.

### Cancellation

The generator kills the `tdl` process when Roslyn cancels the step, and passes no state between runs except what the incremental pipeline caches.

## One generator, two hosts

`dotnet/Tdl.Gen.CSharp` splits into a library holding the generator and the executable serving it as a plugin.
The library, `Tdl.Plugin`, and `Tdl.DotNet` target netstandard2.0 beside net10.0, since an analyzer loads into compilers on .NET Framework and .NET alike.
A polyfill package supplies what the C# 14 syntax they use needs on netstandard2.0, such as `IsExternalInit` and `RequiredMemberAttribute`, and the few runtime APIs missing there are rewritten.

Both hosts reference Roslyn 4.8, the compiler in the .NET 8 SDK, which matches the generated code's net8.0 floor.
An analyzer cannot reference a newer Roslyn than the compiler loading it, and one version in both hosts keeps them from diverging.

A test like `TestHostsAgree` holds the generator to the plugin's bytes for every conformance case and the smoke model, after stripping `#line`.

## Packaging

The generator ships as the NuGet package `Tdl.Generator`, an analyzer under `analyzers/dotnet/cs` with `Tdl.Plugin`, `Tdl.DotNet`, and Google.Protobuf beside it, and `buildTransitive` props declaring `Tdl` and `TdlPath`.
A consumer adds the package and a `.tdl` file with a `csharp` block.

release-please owns its version.
`release-please.yml` gains a job after the reusable release job, run only when that job reports a release: it packs the package and pushes it to nuget.org under trusted publishing, exchanging the job's GitHub OIDC token for a short-lived API key with `NuGet/login`.
The job asks for `id-token: write`, and the nuget.org policy, on the owner's account, names `release-please.yml`.
No API key is stored.
If the reusable workflow in `unmango/actions` exposes no release-created output, it gains one.

Nix builds no package for it.
`checks.dotnet-sdk` builds the generator and runs its tests with the rest of `dotnet/`.

## Testing

Tests drive the generator with `CSharpGeneratorDriver` over the conformance corpus and the smoke model, with `tdl` on `PATH` as CI already arranges for `tdl-gen-csharp`, and compile the result with warnings as errors.
They assert on the diagnostics for a missing `tdl`, a version mismatch, a lowering error, and duplicate output.
Step tracking asserts that a C# edit reruns no step that calls `tdl`, and that editing one `.tdl` file reruns it once.

## Out of scope

- **F#.** F# has no source generators; its equivalent is a type provider, which [#1311](https://github.com/UnstoppableMango/tdl/issues/1311) records.
- **Other targets.** Only C# compiles through Roslyn.
- **A long-lived `tdl`.** Added only if a process per change proves slow.
- **Bundled binaries.** Per-platform `tdl` binaries inside the package, if users ask for them.

## Open questions

- **Rider and unsaved buffers.** Visual Studio passes an unsaved additional file's text to generators; whether Rider and the C# Dev Kit do decides how live the feature is there, and only saving helps where they do not.
- **A model across projects.** Two projects generating from one `.tdl` file each get their own copy of its types, as two `tdl gen` targets would; an extern through a project reference is the remedy, and is not designed here.
