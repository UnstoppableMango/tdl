# The plugin protocol

Design document.
Nothing here is implemented.
It depends on [ir.md](ir.md), which is also unimplemented.

A backend turns a resolved model into files.
`go`, `ts`, and `sql` ship with `tdl`; any other target name resolves to `tdl-gen-<name>` on `PATH`.
Both kinds speak this protocol.

## One protocol, two hosts

A built-in backend is a Go function from request to response, called in-process.
A plugin is an executable exchanging the same messages over stdio.

There is no richer interface available to built-ins.
The backends maintained in this repository exercise exactly the surface a third-party plugin has, which is the only reliable way to keep that surface honest.
When the Go backend needs something the protocol cannot express, the protocol is what changes.

## Transport

Length-prefixed protobuf messages over stdin and stdout, both directions: a varint byte count followed by the encoded message.

Not protoc's model of one request in, one response out, closed stdin.
A stream carries the handshake and the request as ordinary messages on one connection, allows a plugin to be reused across regenerations, and leaves room to move to gRPC later without a second mechanism.

The handshake declares the framing version.
A future `tdl` can offer gRPC, an old plugin can decline, and both continue on length-prefixed stdio.

## Handshake

`tdl` sends first:

- framing version
- ir schema version
- whether this is a watch session

The plugin replies:

- accept, or refuse with the version it needs
- its own name and version
- directives it understands, each with arity and expected literal kinds
- optional features it supports: reuse across requests, and any later additions

A refusal is a clean, readable failure naming both versions.
The alternative, a plugin silently ignoring protobuf fields it was compiled before, produces subtly wrong generated code with no diagnostic anywhere.

### Directive declarations

The plugin declares directives by name, arity, and literal kind: `tag` takes one string, `slice` takes none.

`tdl` checks every declared directive in the target block against that shape and reports `tag 42` with the position in the `.tdl` file, before generating anything.

A directive the plugin did not declare is a warning, not an error, and is passed through anyway.
Under-declaring is a plugin bug that should not break a working project, and a plugin is free to handle more than it advertises.
The warning names the directive and its position so a typo is still visible.

## Request

One message carries everything:

- the target name being served
- the resolved model, one package, per [ir.md](ir.md)
- resolved directives, already attached to the nodes they apply to
- the output path
- a dry-run flag

Nothing travels in argv or environment variables.
A single versioned message is one thing to keep compatible, and a plugin never has to reconcile two channels that disagree.

## Response

The plugin returns file contents, not files on disk:

```proto
message Response {
  repeated File   files       = 1;
  repeated string post        = 2;
  repeated Diagnostic errors  = 3;
}

message File {
  string path    = 1;  // relative to out
  bytes  content = 2;
}
```

`tdl` writes them.
That is what makes `--verify`, `--clean`, the `.tdl-output` marker, and atomic writes enforceable rather than a convention plugins are asked to honor.

Paths are relative to the output directory.
An absolute path, or one containing `..`, is an error and nothing is written.
A plugin cannot reach outside the directory the project pointed it at, whatever it intends.

## Post-processing

A plugin may ask for a command to be run over the written files, typically a formatter.

The commands `tdl` will run are declared in `tdl.toml`:

```toml
[post]
gofmt = "gofmt -w"
```

The plugin requests `gofmt` by name and never supplies arguments.
An undeclared name is skipped with a warning saying what to add to opt in.

Output is still written and still correct, just unformatted, so a fresh checkout of a project generates successfully before its `tdl.toml` is complete.
The plugin is already arbitrary code on `PATH`; the allowlist is not a security boundary, it is a record of what a build runs.

## Dry run

`tdl gen --verify` sets a dry-run flag in the request.

The plugin still returns file contents, and `tdl` diffs them against disk instead of writing.
The flag exists so a backend that would shell out or do expensive work it knows cannot affect the answer can skip it.

A plugin that ignores the flag entirely is correct, only slower.

## Diagnostics

The response carries structured diagnostics: a message, a severity, and the `ir` node ID it concerns.

`tdl` maps the node back to its source position and prints it in the same format as its own errors, so `--format json` covers plugin errors without the plugin implementing anything.

If a plugin exits non-zero or dies before sending a response, `tdl` relays its stderr verbatim and names the plugin.
Structured is the contract; stderr is what is left when the contract could not be met.

## Reuse and watch mode

By default one process serves one generation and exits.

A plugin that declared reuse in the handshake stays alive under `--watch` and receives another request on the same stream.
It must treat each request as independent, and carrying state between them is a plugin bug.

`tdl` restarts a reused plugin when its binary changes on disk, so developing a plugin does not require killing the watch.

## Timeouts

A plugin that never responds is killed after a default timeout, and the error names which plugin hung.

```toml
[plugins]
sql = { command = "tdl-gen-sql", timeout = "5m" }
```

A legitimately slow backend raises its own timeout.
The default exists so a hung plugin in CI fails with a diagnosis rather than consuming the job's entire time budget and reporting nothing.

## Deferred

- gRPC. The handshake carries a framing version so the upgrade is additive; nothing else about it is specified yet.
- Incremental generation. A plugin regenerating only what changed needs a change description in the request, which needs `ir` diffing, which does not exist.
- Plugin-supplied prelude or model contributions. Backends consume the model; nothing lets them extend it.
