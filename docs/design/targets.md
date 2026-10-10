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
| A field's number on the wire, from field order unless pinned | protobuf, thrift |

How each is spelled in a language stays in the target, and a convention a target applies by default writes it with no entry at all.

## Presence and access

### Presence

`T` is present and `T?` may be absent; this exists and stays.
A collection may always be empty, so `[T]`, `{T}`, and `{K -> V}` are absent-able without `?`, and `where { length(1..) }` makes one required.

### Access

The model needs to say that a field is set by the system and never by a writer, or set once at creation.
How it says so is undecided, and so is how it relates to `owned`, which is being reconsidered alongside it.
A keyword or contextual modifier is not the answer: presence and identity are both prelude declarations the compiler knows by name, and access should be the same kind of thing.

apis shows how far access already follows from structure.
Of its 1,155 `OUTPUT_ONLY` fields, 650 are in the `50+` observed band, which holds 651 fields, and 450 are five identity fields every resource repeats: `uid`, `generation`, `create_time`, `update_time`, and `delete_time`.
So access belongs to groups of fields more than to fields, and the remaining 55 sit in the assigned band and declared state.

Two candidates use only existing syntax:

- **A prelude type**, as `T?` is `Option<T>`.
  The prelude declares `Derived<T>` for a field the system sets and `Fixed<T>` for one set once, and backends recognize them by name the way they recognize `Option`.
  A replacement prelude may change what they mean or drop them.
  A mixin carries them with its fields, so apis writes its identity band once:

  ```tdl
  mixin Meta {
    name: string
    uid: Derived<string>
    create_time: Derived<instant>
  }

  type Commit: Entity {
    include Meta
    message: Fixed<string>
    summary: Derived<string>
  }
  ```

- **A group the target names.**
  Access stays out of the model, and a convention maps a class or mixin to an access, so a field arriving through `include Observed` is output-only.
  This writes nothing per field, but the observed band differs on every kind, so apis would need a mixin per kind used once, and every backend's convention would have to agree.

The first keeps access in the model for every backend; the second keeps the model unchanged.
Neither infers access from position: apis's bands are its own convention, and a model without them would get nothing.
Whatever the spelling, the conventions below read three facts per field: whether it may be absent, who writes it, and whether it changes after creation.

### Conventions

Each backend states a convention for writing presence and access, and reads it from a block-scope directive so a project picks one:

| Backend | Directive | `T?` | `T` | output-only | set once |
| --- | --- | --- | --- | --- | --- |
| protobuf | `field_behavior("google")` | `OPTIONAL` | `REQUIRED` | `OUTPUT_ONLY` | `IMMUTABLE` |
| jsonschema, openapi | on by default | not in `required` | in `required` | `readOnly` | none |
| typescript | on by default | `?:` | `:` | `readonly` | `readonly` |

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

### Numbers come from field order

By default a member's number is its place in the declaration: the first field is 1, the next 2.
Numbering in the model is opt-in, for a model whose numbers cannot follow its order.

A field or variant may be prefixed with a number, which pins it, and each unpinned member after it continues from the member before it.
Abridged from apis's `Commit`:

```tdl
type Commit: Entity {
  reserved 2, 3, 6..7, 9..10
  revision: string // 1
  labels: {string -> string} // 4, past the reserved 2 and 3
  annotations: {string -> string} // 5
  commit_time: instant // 8
  repository: Repository // 11
  parent_revisions: [string] // 12
  50 summary: string
  kind: CommitKind // 51
}
```

The reserved numbers do the identity band's work, and the one pin starts the derived band.

Today an unpinned member takes the lowest number no other member holds, so a pin anywhere fills the gaps before it.
Continuing from the previous member keeps a group of fields together after one pin, which is what a model with numbered bands needs: apis's 3,481 numbered type fields would need at most 156 pins, where lowest-free allocation needs about 900 and the target blocks write all 3,481.
A model with no pins numbers exactly as today.
A model that pins out of order may number differently, so the change ships as a `fix!`.

A leading integer cannot begin any other member, so one token decides it.
`reserved` followed by a number or a string lists numbers and names that no member may take, and allocation skips them; it is contextual, so `reserved: bool` is still a field.
A `///` doc comment on a `reserved` member says why, and backends carry it.

Lowering checks that no two members share a number and that none takes a reserved one.
Both are errors in `tdl check`, so they surface while editing rather than as a backend warning that skips the declaration (#991).

Reordering unpinned fields renumbers them, which breaks the wire format.
A project that publishes a schema pins what it has shipped, or relies on its target's own check, such as `buf breaking`.
A target `number` directive on a member the model pins is an error: one fact, one home.
The `number` and `reserved` directives stay for models that number in the target.

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

`=>` applies exactly one directive.
Several use the nested block the grammar already has:

```tdl
target protobuf for unmango.vcs.commit.v1alpha1 {
  edition("2024")
  Commit.tree_uri {
    number(18)
    option("(buf.validate.field).string.uri", "true")
  }
  Fn => rpc
}
```

The trap in #922 is `path => a(...) b(...)`, which reads as two directives on `path` but attaches `b` to the enclosing block, since a block-scope directive may appear anywhere in it.
`tdl fmt` moves every block-scope directive above the first entry of its block, in source order.
That changes no meaning, because a block-scope directive applies to its whole block wherever it is written, and it moves `b` to where it visibly applies, so the mistake shows in the diff and `tdl fmt --check` fails on it.

## Editors

The model and its target blocks stay apart, so an editor shows them together (#992):

- Hover on a field, variant, or declaration lists the directives each target block in scope resolves onto it, per target.
- Inlay hints show a member's number where the model does not write one and a backend allocates it.
- Go to definition on each segment of a target path, and on a selector, which lists every node it matches.
- A code action on an unpinned member pins the number it has now, before a reorder would move it.

## Out of scope

- Imports and `out` against a project root (#995) belong to the `tdl.toml` work in [workflow.md](workflow.md).
- Ordinary `//` comments in generated output (#996); `reserved` docs cover the case apis loses most.
- `foreign` mappings repeated in every importer, which is #916.

## Open questions

- **Any-entity references.** `refs: [Entity]` would say "a reference to some entity" if a class could be a field type. It cannot today; until it can, `"*"` stays an explicit option.
- **Import cycles.** Two domains referencing each other's entities import each other. Whether that is allowed decides whether typed references can replace `ref.ObjectReference` everywhere apis uses it.
- **Variants.** A number on a fielded variant numbers the variant in a oneof; whether its fields number from 1 again or continue the enclosing message's numbers is a protobuf question the backend answers today, and the syntax should not settle it.
- **Spelling access, and `owned`.** See [Access](#access). Undecided, and to be settled together with whether `owned` stays a field modifier; if access becomes a prelude type, composition may be one too.
- **Input-only fields.** A field a writer sets and never reads back (`INPUT_ONLY`, `writeOnly`) has no use in apis, so neither candidate spells it yet.
