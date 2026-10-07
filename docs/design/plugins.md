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

A stream, rather than protoc's one request in and one response out, carries the handshake and the request as ordinary messages, lets a plugin serve several requests, and leaves room for gRPC later.
The handshake declares the framing version, so a future `tdl` can offer gRPC and an old plugin can decline.

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
- directives it understands, each with arity and expected literal kinds
- optional features it supports, such as reuse across requests and import

A refusal is a readable failure naming both versions, where a plugin silently ignoring fields it was compiled before would produce wrong code with no diagnostic.

### Directive declarations

The plugin declares directives by name, arity, and literal kind: `tag` takes one string, `slice` takes none.
The kinds are `ir.LiteralKind`, the set a constraint argument uses: string, int, float, bool, name, regex, list, and range.

`tdl` checks every declared directive in the target block against that shape and reports `tag 42` with its position before generating anything.

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

Nothing travels in argv or environment variables, so there is one versioned message to keep compatible.

### What the model contains

**The prelude.** It is merged into the declaration table untagged, which is what lets a replacement prelude change what a collection is without any backend knowing. A model arrives with every prelude declaration beside its own. Filter by the filename in each declaration's position, as `emit.Own` does.

**Directives for every target block.** They are attached to the nodes they apply to, resolved, with the specificity ladder applied, and each carries the name of its block. A plugin keeps the ones whose target matches its request.

**A directive expanded from a class names the class.** `Auditable => trigger("touch")` reaches every declaration satisfying `Auditable` with `from_class` set, so a backend can say why a rule is there.

Run `tdl ir --format json` over a model to see what a plugin receives.

## Response

The plugin returns file contents, not files on disk:

```proto
message Response {
  repeated File       files       = 1;
  repeated string     post        = 2;
  repeated Diagnostic diagnostics = 3;
}

message File {
  string path    = 1; // relative to out
  bytes  content = 2;
}
```

`tdl` writes them, which is what makes `--verify`, `--clean`, and the `.tdl-output` marker enforceable.

An absolute path, or one containing `..`, is an error and nothing is written, so a plugin cannot reach outside its output directory.

## Post-processing

Not built: nothing reads `Response.post`.

A plugin may ask for a command to run over the written files, typically a formatter.
The commands are declared in `tdl.toml`:

```toml
[post]
gofmt = "gofmt -w"
```

The plugin requests `gofmt` by name and never supplies arguments.
An undeclared name is skipped with a warning saying what to add, and the output is still written, unformatted.
The allowlist is a record of what a build runs, not a security boundary, since the plugin is already arbitrary code.

## Dry run

`tdl gen --verify` sets the dry-run flag.
The plugin still returns file contents, and `tdl` diffs them against disk.
A backend may skip expensive work it knows cannot affect the answer; one that ignores the flag is correct, only slower.

## Import

A handshake in import mode is followed by an `ImportRequest` rather than a `Request`: the target name, the source files, and the loss codes the user allowed.
The plugin answers with an `ImportResponse`, a model and diagnostics, and `tdl` prints the model as TDL.
A plugin declares the `reverse` feature to receive one; `tdl` sends no request to a plugin whose reply lacks it, and a plugin that does not import refuses the mode.
In Go, such a backend also implements `plugin.Importer`.
[reverse.md](reverse.md) is the design.

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

**A dependency's declaration-level directives.** An `ir.Import` carries the block-scope directives of its dependency's target blocks, and nothing beneath them; see [ir-plan.md](ir-plan.md) phase 8b.

**Class-scoped directives on instantiated types.** A class path expands across the declarations satisfying the class, not across types that satisfy it only through a conditional instance: given `instance <T> Auditable<Page<T>>`, a directive on `Auditable` reaches `Audited` and not `Page<Audited>`. `SatisfyingTypes` has the answer, and target resolution does not read it.

## Deferred

- gRPC. The handshake carries a framing version so the upgrade is additive.
- Incremental generation, which needs a change description in the request and therefore `ir` diffing.
- Plugin-supplied prelude or model contributions.
