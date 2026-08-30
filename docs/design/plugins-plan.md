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

**Done.** `plugin.Conn` carries several messages on one connection, distinguishes a stream that ends between messages from one cut part way through, and checks a length prefix before trusting it.

## Phase 2: the backend contract and a first backend

The Go type a backend implements: request in, response out, plus whatever the handshake needs it to declare.

The first backend is `debug`, which emits one file describing the model it was given. It is not useful output, and that is the point: it exercises every part of the protocol without needing anyone to agree on what generated Go should look like.

Done when `debug` produces a response from a lowered model, and the same backend value satisfies the interface the host will call and the SDK will serve.

**Done.** `plugin.Backend` is two methods, `Describe` and `Generate`, and `backend/debug` implements them. `plugin.Directives` filters a node's directives by target, since a model carries every block's.

## Phase 3: `tdl gen`, in-process

The command, wired to the built-in registry only. It lowers, picks the target blocks to run, calls each backend, and writes what comes back.

Writing is where the rules from plugins.md land: paths are relative to the output directory, an absolute path or one containing `..` is an error and nothing is written, and `out` comes from the target block's own directive with `-o` overriding it.

Done when `tdl gen` runs `debug` over a corpus model and writes the file, a path escaping the output directory is refused with a diagnostic, and `--target` narrows a run.

**Done.** Writing checks every path before writing any file, so a response with one bad path writes nothing rather than half of itself.

## Phase 4: `--verify` and `--clean`

The dry-run flag in the request, the diff against disk, and the `.tdl-output` marker that makes `--clean` refuse a directory it does not own.

Done when `--verify` exits non-zero on a stale output directory and writes nothing, `--clean` removes what a previous run wrote, and `--clean` on a directory without the marker is an error rather than a deletion.

**Done.** `--verify` also reports a file tdl wrote and would no longer write, which is what catches a deleted declaration, and only inside a directory carrying the marker: without one there is no way to tell a stale file from a file that was never tdl's.

## Phase 5: the subprocess host

Exec `tdl-gen-<name>` from `PATH`, exchange the handshake, send the request, read the response, and surface a non-zero exit with the plugin's stderr attached.

This is where the protocol first has two hosts, so it is where the parity tests from above start running.

Done when the `debug` backend compiled as `tdl-gen-debug` produces byte-identical output to the built-in through `tdl gen`, a version mismatch is a readable refusal naming both versions, and a plugin that never answers is killed with a diagnostic naming it.

**Done, and it absorbed phase 6.** A host cannot be tested without something serving the other end, so `plugin.Serve` and `cmd/tdl-gen-debug` arrived here rather than after. Phase 6 is left holding what it should have been about: making the SDK worth handing to someone.

## Phase 6: the SDK, documented

`plugin.Serve` and `cmd/tdl-gen-debug` landed in phase 5, because a host with nothing on the other end cannot be tested. What is left is what makes the package worth handing to someone: a written example of a backend, the recorded request and response pairs in `testdata/plugin/`, and the failure paths a plugin author will hit spelled out rather than discovered.

Done when someone can write a backend from the package documentation alone, and the recorded pairs replay against a non-Go implementation.

**Done, except the last clause.** There is no non-Go implementation to replay against yet, so what the recorded pairs prove today is that the Go one is stable. They are written in a form that does not need this repository to interpret, which is the part that had to be got right now.

## Phase 7: directive declarations and diagnostics

The handshake's directive list, checked against the target blocks before generating: a declared directive used with the wrong arity or argument kind is an error with the position in the `.tdl` file, and an undeclared one is a warning that passes through.

Response diagnostics carry a position and print in `tdl`'s own format, so `--format json` covers them.

Done when a target block calling `tag(42)` fails before any backend runs, an undeclared directive warns and still reaches the backend, and a diagnostic from a plugin is indistinguishable in output from one the compiler raised.

**Done.** `out` turned out to need exempting: it is tdl's own directive, read before a backend is involved, so asking a backend to declare it would have made every target block warn.

## Phase 8: reuse and watch

A plugin that declares reuse stays alive across regenerations and receives another request on the same stream. `tdl gen --watch` regenerates on save, and restarts a reused plugin when its binary changes.

Done when a reused plugin serves two requests on one connection, a plugin that does not declare reuse gets a fresh process per generation, and replacing the binary under `--watch` picks up the new one.

**Done.** Watching compares contents rather than modification times, so a save that rewrites the same bytes is ignored.

## Decisions deferred

Points where the implementation picked something reasonable rather than
stopping to settle it. Each is cheap to change while there is one
implementation of the protocol and expensive afterwards.

- **Framing version is an integer.** A string would carry more meaning and
  invites parsing. One is enough until there is a second framing.
- **`MaxMessageSize` is 64 MiB and not configurable.** It bounds the
  length prefix, which is read from a stream before anything is known
  about it. A real model that exceeds it is the signal to revisit.
- **`DirectiveSpec.arg_kinds` constrains by position, and an empty list
  accepts anything.** A directive whose third argument may be one of two
  kinds cannot say so. A repeated-kinds-per-position shape would, at the
  cost of a message nobody needs yet.
- **`Response.post` is a list of names with no arguments.** The project
  decides what a name maps to. Whether a backend should be able to pass
  arguments is a question for whoever implements the allowlist.
- **A backend returns an error only when it cannot produce a response.** A
  problem with the model goes in `Response.diagnostics`, where it reaches
  the user with a position. Nothing enforces the distinction, and a
  backend that returns an error for a bad model still fails usefully, just
  without a position.
- **The protos moved to editions 2024 and `buf breaking` uses FILE.** The
  move is wire-compatible, and `buf breaking` with WIRE_JSON passes clean,
  but at FILE level it reports the syntax change, the `go_package` move
  into managed mode, and twenty-one fields whose C++ string default became
  VIEW. The strict category stays, so the conversion costs one red Buf
  check on the pull request that makes it.
- **Watching polls once a second rather than using an OS notification
  API.** No dependency, same behaviour everywhere, and fast enough for a
  person saving a file. It also means a poll can land mid-save: an editor
  that truncates before writing briefly presents an empty file, which
  reads as a change and fails to parse until the next poll. Editors that
  save by renaming never show it.
- **A restart is detected by the binary's modification time.** Rebuilding
  to an identical binary with a new timestamp restarts the plugin for
  nothing, which is cheap and the wrong way round from missing a real
  change.
- **`out` is the only directive tdl reads itself.** More may follow, and
  the exemption list is a map in `internal/gen/check.go` rather than
  anything the protocol states. A backend cannot find out which names are
  reserved.
- **A recorded exchange is compared by parsing rather than byte for byte.**
  `prototext` output is deliberately unstable across builds, so a byte
  comparison would fail on a toolchain bump. That makes the files less
  useful as a fixture for a parser that does not have protobuf text
  support.
- **An empty output directory is adopted rather than refused.** There is
  nothing in it that could belong to anyone else. A directory that exists,
  is empty, and is meant to stay that way has no way to say so.
- **`--verify` and `--clean` cannot be combined.** One writes nothing and
  the other deletes, so together they would mean "delete and then check
  what I deleted". The command refuses rather than picking.
- **`debug` finds the prelude by looking for a file named `std.tdl`.** A
  model does not say which of its declarations were merged in from a
  prelude. Every backend that generates per declaration needs to know, so
  this is a gap in `ir` that a string match is papering over.

## Deferred

- **Post-processing.** The `[post]` allowlist lives in `tdl.toml`, which nothing reads. The response field can carry the request from phase 1; honouring it waits for the manifest.
- **gRPC.** The handshake carries a framing version so the upgrade is additive. Nothing else about it is specified.
- **Incremental generation.** Needs a change description in the request, which needs `ir` diffing, which does not exist.
- **Real backends.** Go, TypeScript, and SQL are each a separate plan. `debug` exists to keep this one honest, not to be useful.
