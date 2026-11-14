# The plugin protocol

Design document.
The protocol is implemented: the wire types are in `plugin/`, generated from `proto/tdl/plugin/v1/plugin.proto`, and the compiler side is `internal/gen`.
The `tdl.toml` settings below, post-processing commands and per-plugin timeouts, are not built.

A backend turns a resolved model into files.
The backends in `internal/gen/registry.go` ship with `tdl`; any other target name resolves to `tdl-gen-<name>` on `PATH`.
Both kinds speak this protocol.

## One protocol, two hosts

A built-in backend is a Go function from request to response, called in process.
A plugin is an executable exchanging the same messages over stdio.

Built-ins get no richer interface, so the backends in this repository exercise exactly the surface a third-party plugin has.
When a backend needs something the protocol cannot express, the protocol changes.

## Transport

Length-prefixed protobuf messages over stdin and stdout, in both directions: a varint byte count followed by the encoded message.

A stream, rather than protoc's one request in and one response out, carries the handshake and the request as ordinary messages and lets a plugin serve several requests.
The handshake declares the framing version, which is 1 and describes this framing alone.

A plugin may instead choose gRPC over a Unix domain socket, by dialing the socket the host names in `TDL_PLUGIN_SOCKET` rather than answering on stdout.
The messages are the same; [grpc-transport.md](https://github.com/UnstoppableMango/tdl/blob/fc0682d/docs/design/grpc-transport.md) is the design.

## Handshake

`tdl` sends first:

- framing version
- ir schema version
- whether this is a watch session
- the mode: generate, or import for `tdl import`

The ir schema version is the protobuf package, `tdl.ir.v1`.
Within a version, field numbers are never reused and new fields are additive, so a plugin built against an older `v1` keeps working.

The plugin replies:

- accept, or refuse with the version it needs
- its name and version
- directives it understands, each with arity, expected literal kinds, placements, and allowed values
- how it numbers fields and variants, for a wire format that identifies them by number
- optional features it supports, such as reuse across requests, import, and nondeterminism (see [Nondeterministic backends](#nondeterministic-backends))

A refusal is a readable failure naming both versions, where a plugin silently ignoring fields it was compiled before would produce wrong code with no diagnostic.

### Directive declarations

The plugin declares directives by name, arity, and literal kind: `tag` takes one string, `slice` takes none.
The kinds are `ir.LiteralKind`, the set a constraint argument uses: string, int, float, bool, name, regex, list, and range.

A directive may also declare where it is written (the target block, a declaration, a field, a variant, or an extern) and the values its first argument may take: `namespace` belongs on the block, and `draft` takes `"2020-12"` or `"draft-07"`.
An empty list of either accepts anything, and a directive accepted on a declaration is accepted on an extern.

`tdl` checks every declared directive in the target block against that shape and reports `tag 42`, a `namespace` on a declaration, or a `draft("2019")` with its position before generating anything.

A directive the plugin did not declare is a positioned warning and is passed through anyway, since under-declaring is a plugin bug that should not break a working project.

A plugin may declare a directive repeatable, and then every entry at the same specificity reaches it in source order.
For any other directive, two entries at the same specificity are an error at the second one, before anything is generated.
Lowering keeps every entry at the winning specificity, because only the plugin knows which directives may repeat.

## Request

One message carries everything:

- the target name being served
- the resolved model, one package, per [ir.md](ir.md)
- the output path
- a dry-run flag
- the analysis: each type resolved to the prelude's shapes, and what each declaration names, per [plugin-sdk.md](plugin-sdk.md#analysis)

Nothing in a request travels in argv or environment variables, so there is one versioned message to keep compatible.
The environment says only where to connect: `TDL_PLUGIN_SOCKET`, per [grpc-transport.md](https://github.com/UnstoppableMango/tdl/blob/fc0682d/docs/design/grpc-transport.md#the-environment).

### What the model contains

**The prelude.** It is merged into the declaration table untagged, which is what lets a replacement prelude change what a collection is without any backend knowing. A model arrives with every prelude declaration beside its own. The analysis marks which declarations are the model's own.

**Directives for its target.** They are attached to the nodes they apply to, resolved, with the specificity ladder applied, and each carries the name of its block. The model holds only the request's target block and the directives from it; `analysis.Scope` removes the rest.

**A directive expanded from a class names the class.** `Auditable => trigger("touch")` reaches every declaration satisfying `Auditable` with `from_class` set, so a backend can say why a rule is there.

Run `tdl ir --format json` over a model to see what a plugin receives.

## Response

The plugin returns file contents, not files on disk:

```proto
message Response {
  repeated File       files       = 1;
  repeated string     post        = 2; // deprecated, ignored
  repeated Diagnostic diagnostics = 3;
}

message File {
  string path    = 1; // relative to out
  bytes  content = 2;
}
```

`tdl` writes them, which is what makes `--verify`, `--clean`, and the `.tdl-output` marker enforceable.

An absolute path, or one containing `..`, is an error and nothing is written, so a plugin cannot reach outside its output directory.
Writing a response's files, `--verify`, `--clean`, and the `.tdl-output` marker go through an `os.Root` opened at the output directory, so a symlink inside it pointing out of it is refused too: a write naming one writes nothing, and `--clean` reports a marker entry through one and removes the rest.
An output directory that is itself a symlink opens at its target.
The `--make` dependency file is the user's path, not a backend's, and is written without the root.

### Nondeterministic backends

A backend whose same request may produce different files, such as one that hands the mapping to a language model, sets `Features.nondeterministic` in its handshake (ADR-0002).
For such a target, `tdl gen` hashes the request and records the hash in `.tdl-output`, with how many files it wrote, a hash of each, and the diagnostics the backend reported, as comment lines:

```text
#@ yolo request <sha256> 1
#@ yolo file <sha256> lib/order.ex
#@ yolo diagnostics <base64>
```

The hash leaves out positions, the output path, and the dry-run flag, so moving the model within its file or running from another directory keeps the output.
While a later request hashes the same, the record lists as many files as it says, and every recorded file is unchanged, `tdl gen` keeps the files, reports the recorded diagnostics again, and sends no request; otherwise it sends the request and records anew.
`--verify` never sends the request: a changed request, a target never generated here, an incomplete record, and a recorded file that is missing or edited are stale.
`--clean` always sends it.

## Post-processing

Not built; #1409 builds it.

The project, not the plugin, chooses commands to run over a target's output, typically a formatter.
A target block names them with `post`, and `tdl.toml` maps each name to a command line:

```tdl
target typescript for shop {
  out("gen/ts")
  post("biome")
}
```

```toml
[post]
biome = "biome format --stdin-file-path={file}"
```

A command is a filter over one file: `tdl` writes the content to its stdin and takes its stdout as the new content.
`{file}` is the file's destination path relative to the directory holding `tdl.toml`, and the command runs there, so a tool finds its own configuration as if the file were in place.
Several names run in order, each one's output feeding the next, over every file the target returns.

Post-processing runs before anything is compared or written, so `--verify`, `--clean`, `.tdl-output`, and `--make` see the bytes that land on disk.
A command that exits non-zero or is not found, or a name `[post]` does not declare, is an error, and nothing is written for that target; skipping it instead would make `--verify` depend on the machine.
The table is a record of what a build runs, not a security boundary, since the plugin is already arbitrary code.

`Response.post` is deprecated and ignored.
It once let a plugin request a command by name, but which formatter runs is the project's choice, and the backends that format their output do it in process.

## Dry run

`tdl gen --verify` sets the dry-run flag.
The plugin still returns file contents, and `tdl` diffs them against disk.
A backend may skip expensive work it knows cannot affect the answer; one that ignores the flag is correct, only slower.

## Import

A handshake in import mode is followed by an `ImportRequest` rather than a `Request`: the target name, the source files, the loss codes the user allowed, and whether to report read-back loss (`readback`).
The plugin answers with an `ImportResponse`, a model and diagnostics, and `tdl` prints the model as TDL.
A plugin may return TDL source in place of the model; `tdl` lowers it, and reports source that does not lower as an error.
A plugin declares the `reverse` feature to receive one; `tdl` sends no request to a plugin whose reply lacks it, and a plugin that does not import refuses the mode.
In Go, such a backend also implements `plugin.Importer`.
[reverse.md](reverse.md) is the design.

## Transform

Not built.
A third mode sends a `TransformRequest`, a model, and expects a `TransformResponse`, a model or nothing plus diagnostics, so a project can lint or rewrite its model between lowering and generation.
[transforms.md](https://github.com/UnstoppableMango/tdl/blob/fc0682d/docs/design/transforms.md) is the design.

## Diagnostics

The response carries diagnostics: a message, a severity, a source position, and optionally a code.
A loss warning carries a code such as `lossy.collection`, which `tdl.toml` or `--allow-lossy` can silence; see [reverse.md](reverse.md#loss-codes).
A position rather than an `ir` node ID, because `ir` has several ID spaces and an ID alone does not say which; a plugin copies the position of the node it is complaining about, and `tdl` prints it like its own errors.

If a plugin exits non-zero or dies before responding, `tdl` relays its stderr verbatim and names the plugin.

## Reuse and watch mode

By default one process serves one generation and exits.

A plugin that declared reuse stays alive under `--watch` and receives further requests on the same stream.
It must treat each request as independent.
`tdl` restarts a reused plugin when its binary changes on disk.

## Timeouts

A plugin that does not answer within `gen.DefaultTimeout`, two minutes, is killed, and the error names it.
A per-plugin timeout in `tdl.toml` is not built:

```toml
[plugins]
sql = { command = "tdl-gen-sql", timeout = "5m" }
```

## What a plugin will not see

**A dependency's directives beneath a declaration.** An `ir.Import` carries the block-scope directives of its dependency's target blocks, and an `ir.Extern` the declaration-level ones for the declaration it names; an entry for a field of it, and a transitive dependency's entries, reach nothing. See [ir-plan.md](https://github.com/UnstoppableMango/tdl/blob/fc0682d/docs/design/ir-plan.md) phase 8b.

**Class-scoped directives on instantiated types.** A class path expands across the declarations satisfying the class, not across types that satisfy it only through a conditional instance: given `instance <T> Archived<Page<T>>`, a directive on `Archived` reaches `Audited` and not `Page<Audited>`. `SatisfyingTypes` has the answer, and target resolution does not read it.

## Deferred

- Incremental generation, which needs a change description in the request and therefore `ir` diffing.
- Plugin-supplied prelude or model contributions.
