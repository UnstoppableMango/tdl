# Target blocks at scale

Design document.
[targets-plan.md](targets-plan.md) orders the work.

The spec keeps everything a code generator needs in a `target` block and the model pure.
That rule is right, but at scale it costs more than it should, because the model leaves out facts that every backend needs and a target block can only address one node at a time.
This document moves those facts into the model, lets one target entry reach many nodes, and gives editors what they need to show a model and its targets together.

## Evidence

unmango/apis generates 65 protobuf files from 55 `.tdl` files, about 21,000 lines in all, under tdl 0.3.4.
11,236 of those lines are target blocks.

| What the target blocks repeat | Lines |
| --- | --- |
| `number(n)`, as `field => number(n)` or inside a nested block | 3,558 |
| `option("(google.api.field_behavior)", ...)` | 1,883 |
| of which `"OUTPUT_ONLY"` | 1,155 |
| `option("(google.api.resource_reference).type", ...)` | 169 |
| `foreign(...)` for the same seven shim types | 156 |

Not one field in the model is written `T?`.
Whether a field is optional, required, or set only by the server exists only as a protobuf option, so the JSON Schema, OpenAPI, and TypeScript backends cannot see it.
Every cross-package reference is written `ref.ObjectReference`, with the entity it points at in a quoted string, because the protobuf backend writes a field whose type is an entity as the whole entity message:

```tdl
type Commit: Entity {
  repo: Repo // generated: Repo repo = 1;
}
```

The spec says such a field is a reference and `owned` marks composition, so writing it as an embedded message is a bug, not a missing feature.

A field's number sits in the target block, often hundreds of lines below the field.
`Commit.summary` is declared at line 123 of `commit.tdl` and numbered at line 381.

## What belongs in the model

The model is pure means it says nothing about any one language.
It does not mean every fact a backend reads must live in a target block.

A fact belongs in the model when it is true of the domain whatever the backend, and when more than one backend needs it.
`deprecated` is the precedent: it is in the language "so `tdl check` can report uses of deprecated names and every backend can carry the marker into generated code".

By that test, four facts the apis target blocks restate belong in the model:

| Fact | Backends that read it |
| --- | --- |
| A field may be absent | all, already as `T?` |
| A field is set by the system and never by a writer, or only once | protobuf, jsonschema, openapi, typescript, graphql, smithy |
| A field names an entity rather than containing one | protobuf, jsonschema, openapi, smithy, salesforce, sql |
| A field's number on the wire | protobuf, thrift |

How each is spelled in a language stays in the target, and a convention a target applies by default writes it with no entry at all.

## Presence and access

### Presence

`T` is present and `T?` may be absent; this exists and stays.
A collection may always be empty, so `[T]`, `{T}`, and `{K -> V}` are absent-able without `?`, and `where { length(1..) }` makes one required.

### Access

The model needs to say that a field is set by the system and never by a writer, set by a writer and never read back, or set once at creation.
How it says so is undecided, and so is how it relates to `owned`, which is being reconsidered alongside it.

One candidate is three contextual modifiers beside `owned`, `readonly`, `writeonly`, and `immutable`, with a modifier followed by `{` applying it to a group of fields:

```tdl
type Commit: Entity {
  revision: string readonly
  message: string immutable

  /// Computed by whoever indexes the graph, not part of the hash.
  readonly {
    summary: string
    kind: CommitKind
  }
}
```

A group would also give apis's banner comments (`// Derived.`) a doc comment that reaches generated output.
Whatever the spelling, the conventions below read three facts per field: whether it may be absent, who writes it, and whether it changes after creation.

### Conventions

Each backend states a convention for writing presence and access, and reads it from a block-scope directive so a project picks one:

| Backend | Directive | `T?` | `T` | output-only | input-only | set once |
| --- | --- | --- | --- | --- | --- | --- |
| protobuf | `field_behavior("google")` | `OPTIONAL` | `REQUIRED` | `OUTPUT_ONLY` | `INPUT_ONLY` | `IMMUTABLE` |
| jsonschema, openapi | on by default | not in `required` | in `required` | `readOnly` | `writeOnly` | none |
| typescript | on by default | `?:` | `:` | `readonly` | none | `readonly` |

`field_behavior("google")` writes `(google.api.field_behavior)` and imports its file.
`field_behavior("marked")` writes only for fields carrying `?` or an access, so a project moving to the convention can leave its unannotated fields unannotated.
A field that also carries an explicit `option("(google.api.field_behavior)", ...)` keeps the explicit one, and the backend warns.

## References

### A reference is not a containment

A field whose type is an entity and is not `owned` is a reference.
A backend writes a reference the way its target names one, never as the entity itself.

The protobuf backend reads a block-scope `reference(T)` directive naming the message that carries a reference, such as an `ObjectReference` declared in the model or mapped with `foreign`.
Without one, it writes the reference as a `string` and warns, since protobuf has no reference type of its own.

### Resource names

`resource(type, pattern)` on an entity replaces the quoted text-format option:

```tdl
target protobuf for unmango.vcs.commit.v1alpha1 {
  Commit => resource("unmango.vcs.commit/Commit", "commits/{commit}")
}
```

`singular` and `plural` derive from the type's last segment unless `resource` is given them as third and fourth arguments.
A reference to an entity carrying `resource` gains `(google.api.resource_reference).type` with that type, so this:

```tdl
type Commit: Entity {
  repository: Repository
}
```

generates the field apis writes today by hand, with its `resource_reference`; its `field_behavior` follows once [Access](#access) is settled.

A reference to any entity at all, apis's `resource_reference.type = "*"`, stays an explicit option on a field typed with the reference message.
See [Open questions](#open-questions).

### The cost

Typed references make packages import each other.
apis keeps its domains decoupled today by naming a referenced type only in a string, so moving to typed references makes that coupling visible in the import graph.
That is the point of the change, but it means cross-package imports must stay cheap and must not cycle, and a domain that wants to stay decoupled keeps `ref.ObjectReference`.

## Wire identity

A field's number is its identity on the wire across versions of a schema, the same for protobuf and thrift, and a compatibility promise rather than a generator preference.
It is a fact about the model's history, which makes it the same kind of fact as `deprecated`.

A field or variant may be prefixed with its number:

```tdl
type Commit: Entity {
  1 revision: string
  reserved 2, 3, 6..7, 9..10
  4 labels: {string -> string}
  5 annotations: {string -> string}
  8 commit_time: instant
}

enum CommitKind {
  1 Root
  2 Normal
}
```

A leading integer cannot begin any other member, so one token decides it.
`reserved` followed by a number or a string is a member listing numbers and names that no field may take; it is contextual, so `reserved: bool` is still a field.
A `///` doc comment on a `reserved` member says why, and backends carry it.

Lowering checks that no two members share a number and that none takes a reserved one.
Both are errors in `tdl check`, so they surface while editing rather than as a backend warning that skips the declaration (#991).

An unnumbered member is allocated by the backend as today, so a model that never numbers its fields is unchanged.
A target `number` directive on a member the model numbers is an error: one fact, one home.
The protobuf `reserved` directive likewise stays for models that do not number their members.

## Reaching many nodes

### Field selectors

A path segment may be `*`, matching every declaration in the model or every field of one:

```tdl
target protobuf for unmango.vcs.commit.v1alpha1 {
  *.page_token => option("(buf.validate.field).string.max_len", "1024")
  ListCommitsRequest.* => option("(google.api.field_behavior)", "OPTIONAL")
}
```

The first segment may also be a class or a kind selector from [profiles.md](profiles.md), with a field segment after it: `Entity.conditions` reaches the `conditions` field of every type satisfying `Entity` that has one.
A selector that matches nothing in the model is an error, as a path that names nothing is.

A field entry's specificity is its declaration segment's place on the ladder, a type above a class above a kind selector above `*`, and then a named field above `*`.
So `Commit.page_token` beats `*.page_token`, which beats `*.*`.

[profiles.md](profiles.md) forbids a field segment in a profile because a profile knows no fields.
It may name one after a class, a kind selector, or `*`, since those name no declaration of the model a profile is applied to.

### Several directives on one path

`path => a(...) b(...)` applies both directives to `path` (#922).
Today the second attaches to the enclosing scope, because a block-scope directive may appear anywhere in a block.

A block-scope directive must now come before the first entry in its block.
With that rule, a directive after `=>` is followed by another directive of the same entry until a token begins a new entry: a name followed by `=>`, `{`, or `.`, or a `*`, `@`, or integer.

```tdl
target protobuf for unmango.vcs.commit.v1alpha1 {
  edition("2024")
  Commit.tree_uri => number(18) option("(buf.validate.field).string.uri", "true")
  Fn => rpc
}
```

A block-scope directive after an entry is an error naming the entry it would otherwise have joined.
Existing files that put one late, such as apis's `edition("2024")` after its `foreign` entries, move it to the top of the block once.

## Editors

The model and its target blocks stay apart, so an editor shows them together (#992):

- Hover on a field, variant, or declaration lists the directives each target block in scope resolves onto it, per target.
- Inlay hints show a member's number where the model does not write one and a backend allocates it.
- Go to definition on each segment of a target path, and on a selector, which lists every node it matches.
- A code action on a field with no number adds the lowest free one.

## Out of scope

- Imports and `out` against a project root (#995) belong to the `tdl.toml` work in [workflow.md](workflow.md).
- Ordinary `//` comments in generated output (#996); `reserved` docs, and group docs if [Access](#access) adopts groups, cover the two cases apis loses most.
- `foreign` mappings repeated in every importer, which is #916.

## Open questions

- **Any-entity references.** `refs: [Entity]` would say "a reference to some entity" if a class could be a field type. It cannot today; until it can, `"*"` stays an explicit option.
- **Import cycles.** Two domains referencing each other's entities import each other. Whether that is allowed decides whether typed references can replace `ref.ObjectReference` everywhere apis uses it.
- **Variants.** A number on a fielded variant numbers the variant in a oneof; whether its fields number from 1 again or continue the enclosing message's numbers is a protobuf question the backend answers today, and the syntax should not settle it.
- **Spelling access, and `owned`.** See [Access](#access). Undecided, and to be settled together with whether `owned` stays a field modifier.
