# Implementing the plugin protocol

An implementation plan for [plugins.md](plugins.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds the protocol and one backend that proves it works.
It does not build a Go, TypeScript, or SQL backend: those are code generators, and a code generator is a body of work with its own plan and its own conformance story.

The thing being proved is the claim [plugins.md](plugins.md) opens with: a built-in backend and a subprocess plugin speak the same protocol, and the built-in has no richer surface available to it.
A plan that shipped only the in-process path would be able to state that claim without ever testing it.

`ir` is finished enough to hand over.
What a plugin receives, including the prelude declarations that come with every model, is described in plugins.md and printable with `tdl ir --format json`.

## Layout

```text
proto/tdl/plugin/v1/   # the wire schema, public
plugin/                # generated Go, the codec, and the SDK, public
internal/gen/          # the host: backend registry, subprocess, file writing
```

`plugin` is the package a backend author imports, so it is a compatibility surface like `ir`.
`internal/gen` is the compiler side and changes freely.

## What is not here yet

Two things the protocol refers to do not exist, and each blocks part of a phase rather than the whole plan.

**`tdl.toml`.** [workflow.md](workflow.md) puts plugin declarations, timeouts, and the post-processing allowlist in a manifest nothing reads yet. Phase 7 is where that bites; until then a plugin is found on `PATH` and a timeout is a constant.

**`tdl gen` itself.** The command does not exist. Phase 3 introduces it in its narrowest form, and it grows through the later phases rather than arriving whole.

## Testing

The protocol's one invariant is that both hosts behave identically, so the tests are written against the contract rather than against either host.

A backend used by the tests runs through both paths: called in-process, and compiled into a binary the test execs. Every behavioural test asserts the same result from both. A test that can only be written against one host is a sign the protocol has grown a hole.

`testdata/plugin/` holds recorded request and response pairs as protobuf text format, so a non-Go plugin implementation can replay them.

Diagnostics and failure paths get the same treatment as the happy path: a plugin that returns an error, one that dies mid-response, one that writes a path outside the output directory, and one that never answers.

## Phase 1: the wire schema and framing

`proto/tdl/plugin/v1/plugin.proto` gets `Handshake`, `HandshakeReply`, `Request`, `Response`, `File`, and `Diagnostic`. The request embeds an `ir.Model`.

`plugin/` gets the codec: varint length prefix, then the encoded message, in both directions over an `io.Reader` and `io.Writer`.

Done when a message survives a round trip through the codec, a truncated stream is an error rather than a hang, and a message larger than a sensible cap is refused rather than allocated.

## Phase 2: the backend contract and a first backend

The Go type a backend implements: request in, response out, plus whatever the handshake needs it to declare.

The first backend is `debug`, which emits one file describing the model it was given. It is not useful output, and that is the point: it exercises every part of the protocol without needing anyone to agree on what generated Go should look like.

Done when `debug` produces a response from a lowered model, and the same backend value satisfies the interface the host will call and the SDK will serve.

## Phase 3: `tdl gen`, in-process

The command, wired to the built-in registry only. It lowers, picks the target blocks to run, calls each backend, and writes what comes back.

Writing is where the rules from plugins.md land: paths are relative to the output directory, an absolute path or one containing `..` is an error and nothing is written, and `out` comes from the target block's own directive with `-o` overriding it.

Done when `tdl gen` runs `debug` over a corpus model and writes the file, a path escaping the output directory is refused with a diagnostic, and `--target` narrows a run.

## Phase 4: `--verify` and `--clean`

The dry-run flag in the request, the diff against disk, and the `.tdl-output` marker that makes `--clean` refuse a directory it does not own.

Done when `--verify` exits non-zero on a stale output directory and writes nothing, `--clean` removes what a previous run wrote, and `--clean` on a directory without the marker is an error rather than a deletion.

## Phase 5: the subprocess host

Exec `tdl-gen-<name>` from `PATH`, exchange the handshake, send the request, read the response, and surface a non-zero exit with the plugin's stderr attached.

This is where the protocol first has two hosts, so it is where the parity tests from above start running.

Done when the `debug` backend compiled as `tdl-gen-debug` produces byte-identical output to the built-in through `tdl gen`, a version mismatch is a readable refusal naming both versions, and a plugin that never answers is killed with a diagnostic naming it.

## Phase 6: the SDK

The other side of the wire: a `plugin.Serve(backend)` that reads the handshake, replies with what the backend declares, serves requests, and exits.

Without this, "anyone can ship a backend" means "anyone can reimplement the framing", and the protocol has no second implementation to keep it honest.

Done when `tdl-gen-debug` is a `main` calling `Serve` with the same backend value the built-in registry holds, and the parity tests still pass.

## Phase 7: directive declarations and diagnostics

The handshake's directive list, checked against the target blocks before generating: a declared directive used with the wrong arity or argument kind is an error with the position in the `.tdl` file, and an undeclared one is a warning that passes through.

Response diagnostics carry a position and print in `tdl`'s own format, so `--format json` covers them.

Done when a target block calling `tag(42)` fails before any backend runs, an undeclared directive warns and still reaches the backend, and a diagnostic from a plugin is indistinguishable in output from one the compiler raised.

## Phase 8: reuse and watch

A plugin that declares reuse stays alive across regenerations and receives another request on the same stream. `tdl gen --watch` regenerates on save, and restarts a reused plugin when its binary changes.

Done when a reused plugin serves two requests on one connection, a plugin that does not declare reuse gets a fresh process per generation, and replacing the binary under `--watch` picks up the new one.

## Deferred

- **Post-processing.** The `[post]` allowlist lives in `tdl.toml`, which nothing reads. The response field can carry the request from phase 1; honouring it waits for the manifest.
- **gRPC.** The handshake carries a framing version so the upgrade is additive. Nothing else about it is specified.
- **Incremental generation.** Needs a change description in the request, which needs `ir` diffing, which does not exist.
- **Real backends.** Go, TypeScript, and SQL are each a separate plan. `debug` exists to keep this one honest, not to be useful.
