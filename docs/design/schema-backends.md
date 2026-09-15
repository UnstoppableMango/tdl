# Schema backends

Design document.

Five backends turn a resolved model into a schema language rather than a programming language: `protobuf`, `thrift`, `smithy`, `graphql`, and `typescript`.
TypeScript is on the list because what it emits here is a description of JSON on the wire, the same job the other four do.

They cover the core type definitions: structs in all three kinds, both enum shapes, newtypes, aliases, the primitives, the collections, and optionality.

`protobuf`, `thrift`, `smithy`, and `graphql` are implemented, each in `backend/<name>`.
`typescript` follows the same mapping and is not written.
[go-backend.md](go-backend.md) is the model for each decision below, and where a target has no reason to differ from Go it does not.

## What is shared

`backend/internal/emit` holds everything that does not depend on the target language, so five backends do not each carry a copy that drifts.

- **Ownership.** `Own` returns the declarations the model's own file declared, recognizing the prelude by `prelude.Name` in whole, as [go-backend.md](go-backend.md) describes.
- **Directives.** `Find`, `Text`, and `Block` read directives for the target being served and ignore every other block's.
- **Type resolution.** `Resolve` walks a type reference into a `Ref`: a primitive by name, `List`, `Set`, `Map`, `Option`, `Nullable`, or a named declaration, with aliases expanded. It refuses a type parameter, a unit, an extern, a class, and a type applied to arguments, each with a position. Whether a target has a type for a given primitive is the backend's decision.
- **Diagnostics.** An `UnsupportedError` carries a position, and `Warn` turns it into a warning that does not stop the run.
- **Cascade.** A declaration that cannot be generated is skipped, and `Cascade` then skips every declaration naming it, to a fixed point. Emitting a struct whose field names a skipped declaration produces output referring to something it does not declare, which no target accepts.
- **Names.** `Words` splits a TDL name into words, and `Pascal`, `Camel`, `Snake`, and `ScreamingSnake` join them in each target's convention.

`backend/internal/irtest` builds models by hand for tests, so a failure in a backend's test is the backend's and not lowering's.

## What is not here

Each of these is a positioned warning, and the declaration reaching it is skipped:

- generics
- classes
- units
- externs and `foreign`
- entity keys

Constraints are a warning and the declaration is still emitted, since skipping a constrained newtype would leave every field naming it undeclared.
Validation is its own set of decisions about where a check lives.

`Field.default_value` and `owned` are not read.
A referenced entity is embedded by value in every target.

GraphQL emits output types only.
Input types are a second copy of every type with different rules for unions, and nothing here needs them yet.

## The type mapping

**Warn** means a positioned warning, with the reaching declaration skipped.

| TDL | protobuf | thrift | smithy | graphql | typescript |
| --- | --- | --- | --- | --- | --- |
| `string` | `string` | `string` | `String` | `String` | `string` |
| `int` | `int64` | `i64` | `Long` | `scalar Long` | `number` |
| `bool` | `bool` | `bool` | `Boolean` | `Boolean` | `boolean` |
| `bytes` | `bytes` | `binary` | `Blob` | `scalar Bytes` | `string` |
| `decimal` | `string` | `string` | `BigDecimal` | `scalar Decimal` | `string` |
| `uuid` | `string` | `string` | `String` | `scalar UUID` | `string` |
| `instant` | `google.protobuf.Timestamp` | `string` | `Timestamp` | `scalar DateTime` | `string` |
| `date` | `string` | `string` | `String` | `scalar Date` | `string` |
| `duration` | `google.protobuf.Duration` | `string` | `String` | `scalar Duration` | `string` |

`int` has no width in the spec, so every target uses 64 bits, and GraphQL needs a custom scalar for it because its `Int` is 32.
`decimal` is a string wherever the target has no exact decimal, for the reason [go-backend.md](go-backend.md) gives.
A GraphQL custom scalar is declared only when something uses it.

| TDL | protobuf | thrift | smithy | graphql | typescript |
| --- | --- | --- | --- | --- | --- |
| `List<T>` | `repeated T` | `list<T>` | a `list` shape | `[T!]` | `T[]` |
| `Set<T>` | `repeated T` | `set<T>` | a `@uniqueItems` `list` shape | `[T!]` | `T[]` |
| `Map<K, V>` | `map<K, V>` | `map<K, V>` | a `map` shape | warn | `Record<K, V>` |
| `T?` field | `optional` | `optional` | no `@required` | `T` | `name?: T` |
| `T \| null` field | `optional` | `optional` | no `@required` | `T` | `name: T \| null` |
| `[T?]` | warn | warn | `@sparse` | `[T]` | `(T \| null)[]` |
| `T? \| null` | warn | warn | warn | warn | `name?: T \| null` |

Each target states less than TDL does somewhere.

- Protobuf has no set, so a `Set` is `repeated` and uniqueness is not enforced.
  A repeated or map field cannot be optional or hold another collection, so those shapes warn.
  A map key must resolve to a string, an integer, or a bool.
- Smithy names every collection, so the backend synthesizes one shape per distinct collection type and names it from its element and key.
  A map key must resolve to a string or a fieldless enum.
- GraphQL has no map.
- TypeScript types a map key as `string` or `number`, and one that is a fieldless enum as `Partial<Record<E, V>>`.

A field that is not optional is non-null in GraphQL and `@required` in Smithy.
Thrift fields that are not optional use the default requiredness and never `required`, which Thrift's own guidance advises against.

| TDL | protobuf | thrift | smithy | graphql | typescript |
| --- | --- | --- | --- | --- | --- |
| entity, value, mixin | `message` | `struct` | `structure` | `type` | `interface` |
| fieldless enum | `enum` | `enum` | `enum` | `enum` | a union of string literals |
| fielded enum | a `message` with a `oneof` | a `union` | a `union` | a `union` | a discriminated union |
| newtype | expanded | `typedef` | a named simple shape | expanded | `type N = Base` |
| alias | expanded | expanded | expanded | expanded | expanded |

The three struct kinds emit the same shape, for the reason [go-backend.md](go-backend.md) gives.
A mixin is emitted too, and a struct including it already carries its fields.

A fielded enum is each target's sum type.

- In protobuf, each variant is a nested message and the enum is a message holding a `oneof` of them.
- In Thrift, Smithy, and GraphQL, each variant is a struct named after the enum and the variant, such as `PaymentCard`.
  Smithy targets `Unit` for a variant with no fields.
- A GraphQL object needs at least one field, so a fieldless variant carries a placeholder `_: Boolean` that is always null.
- In TypeScript, each variant is an interface with a `kind` field holding the variant's name. A `discriminant` directive renames the field.

A protobuf enum starts with an `_UNSPECIFIED` value at zero, which proto3 requires, and its values are prefixed with the enum's name, since protobuf enum values share one scope per package.

Protobuf and GraphQL expand a newtype to its base.
A protobuf wrapper message would change the wire format, and a GraphQL custom scalar per newtype would need server code for each one.
A TypeScript newtype is a plain alias rather than a branded type, so parsed JSON needs no cast.

## Names

A declaration is Pascal case in every target.
A protobuf field is snake case and a protobuf enum value is screaming snake case, which is the protobuf style guide.
Every other target writes a field as TDL does.
A `name` directive replaces the name in any target, and a name that collides after conversion, with a keyword, or with a synthesized name is a warning that `name` resolves.

## Numbering

Protobuf fields and Thrift fields carry numbers that are the wire format, and the IR has none.
A field is numbered by its position, from one, and a `number(n)` directive pins it.
Enum values and variants are numbered the same way.

Position is fragile, and pinning is the answer.
Inserting a field anywhere but the end renumbers every field after it.
A mixin's fields are copied into each struct including it, so adding one renumbers every includer.

Protobuf numbers run to 536870911, with 19000 to 19999 reserved by protobuf itself.
Thrift numbers run to 32767.

## Output

Each backend writes one file per model, since a schema language reads a package as one document and cross-file imports would be layout the consumer did not ask for.

| Target | File | Namespace |
| --- | --- | --- |
| protobuf | `<package as directories>/<last segment>.proto` | `package` directive, else the model's package |
| thrift | `<last segment>.thrift` | `namespace *` from the `package` directive, else the model's package |
| smithy | `<last segment>.smithy` | `namespace` from the `package` directive, else the model's package |
| graphql | `<last segment>.graphql` | none |
| typescript | `<last segment>.ts` | none |

Every file starts with `Code generated by tdl. DO NOT EDIT.` in the target's comment syntax.
Doc comments are carried in each target's form, and a deprecation becomes each target's `deprecated` marker, with GraphQL putting a type's deprecation in its description because a type cannot carry `@deprecated`.

## Encodings

TDL defines no wire encoding, so these backends and the Go backend can disagree about one value.
TypeScript types a `duration` and a `date` as strings, and Go's default `encoding/json` writes a `time.Duration` as integer nanoseconds and a date as a full timestamp.
A Go pointer marshals to `null` and TypeScript types `T?` as an absent key.
The `tag` directive is how a Go consumer aligns them, and a wire encoding is a decision for the spec rather than for five backends separately.
