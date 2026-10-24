# TDL

A language for describing domain models, and the compiler that lowers them and hands them to backends.

## Language

### Plugin protocol

**Backend**:
What turns a resolved model into files for one target, or reads files back into a model.
_Avoid_: generator, emitter

**Plugin**:
A backend running as its own executable, found as `tdl-gen-<name>`.
_Avoid_: extension, external backend

**Host**:
The side of the protocol that starts a backend and sends it requests: `tdl` itself, or a test.
_Avoid_: client, driver

**Handshake**:
The first exchange on a connection, where the host states what it will send and the backend says whether it accepts and what it supports.

**Transport**:
The channel and call model requests travel over: stdio, or gRPC.
_Avoid_: framing, when the channel is meant

**Transport selection**:
How a plugin, once spawned, settles on one transport: answering the handshake on stdout picks stdio, and dialing the host's socket picks gRPC.
_Avoid_: upgrade, negotiation

**Framing**:
How one message is delimited on a byte stream: a varint length followed by the encoded message.
_Avoid_: transport, protocol version
