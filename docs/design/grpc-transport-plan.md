# Implementing the gRPC transport

An implementation plan for [grpc-transport.md](grpc-transport.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan adds gRPC over a Unix domain socket as a second plugin transport, picked by the plugin at spawn.
The messages, the handshake's content, and every backend's output stay the same.

A daemon shared across `tdl` processes, transform mode's method, and gRPC in `tdl-gen-csharp` are out of scope.

## Phase 1: the service and the host

- `plugin.proto` gains `service Backend` with `Handshake`, `Generate`, and `Import`; `make generate` adds the gRPC stubs, and `buf.gen.yaml` gains the `go-grpc` plugin.
- `plugin/grpcplugin` holds the server over a `plugin.Backend`, enforcing `Handshake` first and a method matching its mode, and `Serve`, which dials `TDL_PLUGIN_SOCKET` when it is set and calls `plugin.Serve` otherwise.
- `internal/gen` creates the 0700 directory and the socket, sets the variable, writes the handshake to stdin, and takes the first of a stdout reply or a connection.
  A gRPC session implements what the pipe session does, so `Generate` and `Import` do not know which they hold.
- Stdin and stdout stay open as the lifeline; stdout is relayed after a connection, and the timeout covers both waits.
- The socket is not offered when it cannot be created or its path is too long.
- `plugins.md` points here from its transport section.

Done when a test plugin built on `grpcplugin` generates through `tdl gen`, a stdio plugin built before this phase still does, and concurrent `Generate` calls pass under `-race`.

## Phase 2: every Go backend over gRPC

- Each `cmd/tdl-gen-<name>` calls `grpcplugin.Serve`.
- `TestHostsAgree` and `TestImportHostsAgree` gain a gRPC host, so every backend produces the same bytes in process, over a pipe, and over the socket.
- `testdata/plugin/` gains a recorded gRPC exchange for other implementations to replay.

Done when both tests pass for every row of `shipped`, and CI's race run holds them.

## Phase 3: .NET

- `dotnet/Tdl.Plugin.Grpc` serves a `Tdl.Plugin` backend over the service, choosing the transport as `grpcplugin` does.
- The debug port, `tdl-gen-dotnet_debug`, serves through it, and `checks.gen-dotnet-debug` runs it over both transports.
- `tdl-gen-csharp` stays on stdio, since `Grpc.AspNetCore` does not suit Native AOT.
- `deps.json` pins the new packages.

Done when the debug port produces the same bytes over the socket as over stdio.
