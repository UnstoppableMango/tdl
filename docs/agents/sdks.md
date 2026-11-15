# Plugin SDKs

## .NET SDK

`dotnet/` holds `Tdl.Plugin`, the SDK a backend written in .NET builds on, its tests, `Tdl.Plugin.Debug`, a port of `backend/debug` served as `tdl-gen-dotnet_debug`, and the `csharp` backend with `Tdl.DotNet` (see [Backends](backends.md)).
The SDK holds framing, the handshake, a `Session` over the request and its analysis, `Resolve` and constraint sources, diagnostics and loss codes, the cascade, names and case styles, and the round-trip helpers; it computes no fact the analysis carries, and throws on a request without one.
`Tdl.Plugin/Generated/` is `buf generate` output from protoc's C# generator, so `command make generate` needs `protoc`, which the devShell carries; its version must match the `Google.Protobuf` package.
`buf.gen.yaml` leaves out the `pb` package, so the generated descriptors drop the Go options and name neither `go_features.proto` nor `descriptor.proto`.
The tests replay `testdata/plugin/*.binpb` byte for byte and check `LossCodes` against `reverse.md`.
`deps.json` pins every NuGet package for `buildDotnetModule` but those the SDK carries; after changing a package reference, run `command make dotnet-deps`, which needs `dotnet` and the network but not Nix.
It runs `go run ./tools/nugetdeps`, which restores the solution for each runtime the flake builds into one empty folder and writes what nixpkgs' `nuget-to-json` would, laid out as jsonfmt lays it out; `internal/nugetdeps` lists what the SDK carries.
CI runs the same command before the build and fails if `git diff` shows the lock changed, and Renovate runs it on every NuGet update through the `nuget2nix` preset in `UnstoppableMango/renovate-config`, which the-cluster's Renovate allows in `allowedCommands`.
`.github/renovate.json` pins Renovate's SDK to a release of the major `nix/dotnet.nix` builds with, since another major restores other implicit packages; Renovate reads the constraint as an exact version, so it names one.
`checks.dotnet-sdk` builds and tests every project, and `checks.gen-dotnet-debug` runs the debug port through `tdl gen` beside the Go backend.

## Rust SDK

`rust/` is a Cargo workspace holding `tdl-plugin`, the SDK a backend written in Rust builds on, and `tdl-plugin-debug`, a port of `backend/debug` served as `tdl-gen-rust_debug`.
The crates are not published.
The SDK holds framing (`Conn`), the handshake answered from a `Description` (`serve`, `run`), a `Session` over the request and its analysis, diagnostics with loss codes, the cascade (`Skipped`), and case styles over name words.
Like the .NET SDK it computes no fact the analysis carries, and `Session::new` returns an error for a request without one.
A session finds a node's view by the node's address in the request, as the .NET SDK compares references, so a node cloned out of the request has none.
`lossy` reports output loss and `lose` read-back loss, which reports only under `readback` and never under `roundtrip`; round-trip and import helpers wait for a Rust backend that imports.
`tdl-plugin/src/generated/` is `buf generate` output from buffa's `protoc-gen-buffa`, one self-contained file per package (`file_per_package`, `strategy: all`) without views, and leaving out `pb` as the C# entry does; prost and rust-protobuf v3 cannot read Editions 2024.
`buffa` is pinned exactly in `rust/Cargo.toml` to the version `nix/protoc-gen-buffa.nix` builds, since the generated code targets one runtime; bump both together and regenerate, since `.github/renovate.json` leaves `buffa` alone.
The SDK's tests check the loss codes against `reverse.md` and the session, the framing, and the handshake; the debug port's replay `testdata/plugin/*.binpb` byte for byte, in process and over a connection.
`checks.rust-sdk` builds and tests the workspace with no network, and `checks.gen-rust-debug` runs the debug port through `tdl gen` beside the Go backend.

## JS SDK

`js/` is an npm workspace holding `@unmango/tdl-plugin` (`js/tdl-plugin`), the SDK a backend written in TypeScript builds on, and `js/tdl-plugin-debug`, a port of `backend/debug` served as `tdl-gen-js_debug` (#1480).
Nothing in it is published.
It is ESM for Node 22 or later and Bun, using only what both have; `js/biome.json` refuses `bun` and `bun:test` imports.
`tsc` compiles each package's `src/` and `test/` into its `dist/`, and the tests, written against `node:test` and `node:assert`, run from there under both `node --test` and `bun test`.
The SDK holds framing (`Conn`, over any async iterable of bytes and a writable), the handshake answered from a `Description` (`serve`, `run`), a `Session` over the request and its analysis, diagnostics with loss codes, the cascade (`Skipped`), and case styles over name words.
Like the other SDKs it computes no fact the analysis carries, and the `Session` constructor throws a `ProtocolError` for a request without one.
A session reaches the model through methods only, so a later transport can sit behind the same API, and finds a node's view by the node's identity in the request, so a node cloned out of it has none.
`lossy` reports output loss and `lose` read-back loss, which reports only under `readback` and never under `roundtrip`; round-trip and import helpers wait for a JS backend that imports.
`tdl-plugin/src/generated/` is `buf generate` output from protobuf-es's `protoc-gen-es`, run from `js/node_modules` (so `command make generate` runs `npm ci` there first), leaving out `pb` as the C# and Rust entries do.
`@bufbuild/protobuf` and `@bufbuild/protoc-gen-es` are pinned exactly to one version, since the generated code targets one runtime; bump both together and regenerate, since `.github/renovate.json` leaves them alone.
The SDK's tests check the loss codes against `reverse.md` and the session, the framing, and the handshake; the debug port's replay `testdata/plugin/*.binpb` byte for byte, in process and over a connection.
`checks.js-sdk-node` and `checks.js-sdk-bun` build and test the workspace with no network, and `checks.gen-js-debug` runs the debug port through `tdl gen` beside the Go backend.
CI's `check` job runs `command make js-check`.
