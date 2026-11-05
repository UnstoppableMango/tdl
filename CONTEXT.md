# TDL

A language for describing domain models, and the compiler that lowers them and hands them to backends.

## Language

### Models

**Finite value**:
What a struct, an enum, or a newtype must have to be valid: a value built in finitely many steps, so a field of a collection or an optional, or an enum variant that holds no cycle, ends any recursion.
_Avoid_: finite representation, well-founded

### Plugin protocol

**Backend**:
What turns a resolved model into files for one target, or reads files back into a model.
_Avoid_: generator, emitter

**Target**:
The name a target block gives, saying which backend reads the block: the backend registered under that name, or the plugin `tdl-gen-<name>`.
_Avoid_: generator, language, when the name is meant

**Plugin**:
An executable serving one or more backends, started for a target as `tdl-gen-<target>`.
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

### Round trips

**Round trip**:
A conversion through one target and back through that target's importer: model first, a model generated and imported back, or schema first, a hand-written schema imported and regenerated.
"Round-trips" claims the result comes back equal, the model positions aside or the schema under the target's normal form; a trip that does not is described by what it loses.
_Avoid_: round trip, when a value is encoded and decoded or a file printed and reparsed; say what survives

**Annotation**:
A fact written into generated output in the target's own extension syntax, so importing the output recovers what the mapping would otherwise lose; a model-first round trip comes back equal only with annotations.
_Avoid_: round-trip annotation, roundtrip mode

### Loss

**Output loss**:
A fact of the input that generating or importing drops or weakens, so the output states a looser contract than the input does: a skipped declaration, a dropped constraint or default, `uuid` written as a string, or a target construct with no TDL form.
_Avoid_: loss, lossy output

**Read-back loss**:
A fact the output keeps but the opposite direction would not recover: a generated entity reads back as a value, an imported unexported Go type regenerates exported.
_Avoid_: loss, round-trip loss

### Generated types

**Wire convention**:
The one JSON shape several targets read and write for a model's values: properties named as the model spells them, an absent optional left out, and a fielded enum tagged by its discriminant, `kind` unless renamed.
_Avoid_: JSON format, serialization

**Wire compatible**:
What two targets' generated types are when each reads every value the other writes in the wire convention.
_Avoid_: round trip, interop

**Checked type**:
A generated type whose values can only be obtained by passing the checks its constraints imply, so code holding one knows the checks ran: a Rust or Python type built through its constructor, or a Zod brand.
_Avoid_: branded type, refined type, validated type

**Narrowed type**:
A generated type that states a constraint by itself, so a violating value does not type check and no check need run: a union of the allowed literals, or a tuple of the allowed length.
_Avoid_: literal type, refined type
