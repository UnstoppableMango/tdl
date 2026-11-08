# TDL

A language for describing domain models, and the compiler that lowers them and hands them to backends.

## Language

### Models

**Finite value**:
What a struct, an enum, or a newtype must have to be valid: a value built in finitely many steps, so a field of a collection or an optional, or an enum variant that holds no cycle, ends any recursion.
_Avoid_: finite representation, well-founded

**Model fact**:
A fact true of the domain with no target in view: what a thing is, what identifies it, or which values are valid; it belongs in the model, where every target reads it.
When that is unclear, a fact every target must agree on for their outputs to describe the same thing is a model fact.
_Avoid_: metadata, annotation, option

**Target fact**:
A fact about how one target spells, lays out, numbers, or places part of a model; it belongs in that target's block, and the model never names a target.
_Avoid_: option, annotation, metadata

**Convention**:
A named set of facts a group of targets agree on, about an encoding or an API style rather than the domain, written once for every target that follows it: the **wire convention**, Google's resource names.
A fact that every target must agree on only because they share an encoding belongs to a convention, not the model.
_Avoid_: profile, shared target, standard

### Plugin protocol

**Backend**:
What turns a resolved model into files for one target, or reads files back into a model.
_Avoid_: generator, emitter

**Delegated backend**:
A backend that hands the mapping to a language model instead of encoding it, keeping for itself only the prompt, the file paths, the checks it runs on the result, and the loss it reports; the `yolo` target.
_Avoid_: AI backend, LLM backend

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
A fact written in a target's own extension syntax that importing reads back, recovering what the mapping would otherwise lose: generating writes them on request, and a hand-written schema may carry them too; a model-first round trip with read-back loss comes back equal only with annotations.
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
The **convention** for JSON: the one shape several targets read and write for a model's values: properties named as the model spells them, an absent optional left out, an empty list written `[]` and an empty map `{}` at every depth, never `null`, and a fielded enum tagged by its discriminant, `kind` unless renamed.
_Avoid_: JSON format, serialization

**Wire compatible**:
What two targets' generated types are when each reads every value the other writes in the wire convention.
_Avoid_: round trip, interop

**Checked type**:
A generated type whose values can only be obtained by passing the checks its constraints imply, so code holding one knows the checks ran: a Rust or Python type built through its constructor, or a Zod brand.
A `typescript` constructor checks, but its type is not a checked type, since a cast still makes a value without it.
_Avoid_: branded type, refined type, validated type

**Brand**:
A property a generated type is intersected with, keyed by its declaration's qualified name, so the target's compiler tells apart two newtypes over one base: a `typescript` newtype under the `brand` directive. It records no check, so unlike a Zod brand it is not a checked type.
_Avoid_: tag, nominal marker

**Narrowed type**:
A generated type that states a constraint by itself, so a violating value does not type check and no check need run: a union of the allowed literals, or a tuple of the allowed length.
_Avoid_: literal type, refined type

### Tooling

**Skill**:
A folder of instructions and reference files, in the Agent Skills format, that an agent loads to do one task with TDL, such as writing a model or generating code from one; installed in a project or in a user's global skills.
A Claude Code marketplace entry carrying skills is not a **Plugin**.
_Avoid_: plugin, extension, agent plugin

**Skill library**:
The skills this repository publishes for working with TDL, released with `tdl` and named `tdl-<task>`.
_Avoid_: skill pack, plugin
