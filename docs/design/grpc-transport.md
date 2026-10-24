# The gRPC transport

Design document.
Nothing here is built; [grpc-transport-plan.md](grpc-transport-plan.md) orders the work.

[plugins.md](plugins.md) defines one transport: length-prefixed protobuf over stdin and stdout.
This design adds a second, gRPC over a Unix domain socket, which a plugin may choose instead.
The messages do not change: a gRPC plugin receives the same `Handshake`, `Request`, and `ImportRequest`, and answers with the same replies.

## Why

**A plugin from a stub.**
An author in a language with gRPC tooling generates a server from `plugin.proto` and fills in its methods.
They write no framing, no read loop, and no handshake state machine, which today every plugin without an SDK writes by hand.

**Calls in flight together.**
A stdio connection answers one request at a time, in order.
A gRPC server takes concurrent calls by default, so a watch session or the language server can ask one plugin several things at once.

Speed is not a reason: a varint prefix over a pipe is as cheap as framing gets.
A daemon several `tdl` processes share is out of scope; a plugin still lives exactly as long as the host that spawned it.

## Terms

The **transport** is the channel and call model requests travel over: stdio or gRPC.
**Framing** is how one message is delimited on a byte stream, and only stdio has it.
**Transport selection** is how a plugin, once spawned, settles on one: answering the handshake on stdout picks stdio, and dialing the host's socket picks gRPC.
[CONTEXT.md](../../CONTEXT.md) holds these.

## Why not the framing version

`plugins.md` said the handshake's framing version would let a future `tdl` offer gRPC and an old plugin decline.
It cannot: `plugin.ServeConn` and the .NET `Server` refuse any framing version but 1, so an offer of 2 is a failed connection, and falling back would mean starting the plugin again.
`framing_version` stays 1 and describes stdio's framing alone.

## Transport selection

When the host can create the socket, it spawns every plugin the same way:

1. It makes a private directory, mode 0700, and listens on a socket inside it.
2. It starts the plugin with `TDL_PLUGIN_SOCKET` set to the socket's path.
3. It writes the `Handshake` to the plugin's stdin, as it does today.
4. It waits for whichever comes first: a `HandshakeReply` on stdout, or a connection on the socket.

A plugin that answers on stdout has chosen stdio, and the connection proceeds as in `plugins.md`; the host closes the listener and removes the directory.
A plugin that dials has chosen gRPC; the host accepts that one connection, closes the listener, and calls `Handshake` over it.

A plugin built before this design ignores the variable and answers on stdout, so nothing it does changes.
A gRPC plugin never reads stdin, and the handshake the host wrote there is a few bytes sitting in the pipe.
A plugin that speaks both decides for itself; the SDKs dial whenever the variable is set and they were built with gRPC.

When the host cannot create the socket, it leaves the variable unset, and the plugin must answer on stdout.
A plugin that speaks only gRPC then fails with a message saying so, which its SDK writes to stderr.
A socket path longer than the platform allows (108 bytes on Linux, 104 on macOS) counts as cannot.

### The environment

`plugins.md` said nothing travels in argv or environment variables.
The reason was one versioned message to keep compatible: target, output path, dry run, and mode are protobuf fields, never strings on the side.
A socket path is not request content; it says where to connect, as `CSI_ENDPOINT` does for a CSI driver.
The rule becomes: requests travel in messages, and the environment says only where to connect.
`TDL_PLUGIN_SOCKET` is the only variable the host sets.

### Windows

The same socket is used on every platform.
`AF_UNIX` exists on Windows 10 1803 and later, and Go, .NET, and gRPC's `unix:` scheme all support it there.
Where it fails, the host offers no socket, and the plugin uses stdio.

## The service

```proto
service Backend {
  rpc Handshake(Handshake) returns (HandshakeReply);
  rpc Generate(Request) returns (Response);
  rpc Import(ImportRequest) returns (ImportResponse);
}
```

It lives in `plugin.proto`, in `tdl.plugin.v1`, beside the messages it carries.
Every method is unary, which is what a generated server expects; a stream would put back the state machine the stub was meant to remove.
Transform mode, when built ([transforms.md](transforms.md)), adds `Transform` beside them.

`Handshake` is called once, first, on each connection, carrying the same fields as on stdio, mode and watch included.
A refusal, the directive specs, numbering, and features mean what they mean in `plugins.md`, and the host checks the target block against them before any request.
`Generate` before `Handshake`, or a method for a mode the handshake did not name, fails with `FAILED_PRECONDITION`.

### Concurrency and reuse

Choosing gRPC promises reuse and concurrency: the plugin may receive several calls at once and must treat each as independent.
`Features.reuse` is implied rather than read.
A server generated from the service is concurrent unless its author works to make it otherwise.

## Lifetime

Stdio stays open as the plugin's lifeline under either transport.
The host closing stdin means exit, and stdout reaching EOF tells the host the plugin died.
Stderr is relayed as it is today, and so is the timeout: one that covers no answer on stdout and no connection on the socket.
`tdl` restarts a reused plugin when its binary changes, whichever transport it chose.

A gRPC plugin must write nothing to stdout before it connects, because the host is reading stdout for a stdio reply.
Bytes there that do not decode as a `HandshakeReply` are an error naming the plugin, as they are today.
After the connection, the host relays stdout like stderr.

There is no setting to force stdio: a plugin that speaks only gRPC could not honour it, and a plugin that speaks both already chooses.

## Where the code lives

`plugin/` is public and every Go plugin imports it, so gRPC stays out of it.
`plugin/grpcplugin` holds the server and `Serve`, which dials when `TDL_PLUGIN_SOCKET` is set and calls `plugin.Serve` otherwise; a plugin's main is that call.
`internal/gen` holds the host side: the listener, selection, and a gRPC client beside the pipe session.

In .NET, `Grpc.AspNetCore` is large and works poorly with Native AOT, so gRPC goes in a separate `Tdl.Plugin.Grpc` package, and `tdl-gen-csharp` stays on stdio.

## Testing

`TestHostsAgree` holds every Go backend to the same bytes in process and over a pipe.
gRPC becomes its third host: each `cmd/tdl-gen-<name>` is served through `plugin/grpcplugin`, and every request must produce the same response over the socket.
`TestImportHostsAgree` gains the same column.
