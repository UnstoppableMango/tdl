# Schema backends

Design document.

Five backends, each in `backend/<name>`, turn a resolved model into a schema language: `protobuf`, `thrift`, `smithy`, `graphql`, and `typescript`, whose output describes JSON on the wire.

The `salesforce` backend builds on the same pieces and is mapped in [salesforce-backend.md](salesforce-backend.md).
Where a target has no reason to differ from [go-backend.md](go-backend.md), it does not.

## What is shared

`backend/internal/emit` holds everything that does not depend on the target language:

- **Ownership.** `Own` returns the declarations the model's file declared, recognizing the prelude by `prelude.Name` in whole.
- **Directives.** `Find`, `Text`, and `Block` read directives for the target being served and ignore every other block's.
- **Type resolution.** `Resolve` walks a type reference into a `Ref`, with aliases expanded. It refuses a type parameter, a unit, a class, a type applied to arguments, and an extern unless the session sets `Externs`, each with a position.
- **Diagnostics.** An `UnsupportedError` carries a position, and `Warn` turns it into a warning that does not stop the run.
- **Cascade.** `Cascade` skips every declaration naming a skipped one, to a fixed point, since no target accepts a reference to something the output does not declare.
- **Names.** `Words` splits a TDL name into words, and `Pascal`, `Camel`, `Snake`, and `ScreamingSnake` join them.

`backend/internal/irtest` builds models by hand for tests.

## What is not here

Each of these is a positioned warning, and the declaration reaching it is skipped:

- generics
- classes
- units
- externs and `foreign`, except in protobuf
- entity keys

Protobuf reads `foreign(file, message)` on a declaration or an extern as a message another proto file declares.
The declaration is not emitted, every reference is written as `message`, and each output file referencing it imports `<file>` once.
An extern without `foreign`, whose dependency has a protobuf target block, is the message that dependency generates: it is written `<package>.<Name>` and imported from `<package dirs>/<file>`.
The package is the block's `package` directive or else the dependency's package, the file is its `file` directive or else the package's last segment with `.proto`, and the name is Pascal-cased, so the path matches the dependency's output.
An extern with neither warns.

Protobuf declares services with three argument-less directives.
A structure tagged `service` is emitted as a `service`, and each of its fields is typed by a primitive tagged `rpc` taking the request and response: the field becomes `rpc <Field>(<Request>) returns (<Response>);`.
A primitive tagged `stream` wraps a request or response, as in `rpc Chat(stream Chunk) returns (stream Widget);`.
Either primitive may be declared in the file or imported.
The request and response must be message references, and a service field that is not an rpc warns and skips the service.

Constraints warn and the declaration is still emitted.
`Field.default_value` and `owned` are not read, and a referenced entity is embedded by value.

GraphQL emits output types only; input types would be a second copy of every type with different union rules.

## The type mapping

**warn** below means a positioned warning, with the reaching declaration skipped.

| TDL | protobuf | thrift | smithy | graphql | typescript |
| --- | --- | --- | --- | --- | --- |
| `string` | `string` | `string` | `String` | `String` | `string` |
| `int` | `int64` | `i64` | `Long` | `scalar Long` | `number` |
| `int32` | `int32` | `i32` | `Integer` | `Int` | `number` |
| `uint32` | `uint32` | `i64` | `Long` | `scalar Long` | `number` |
| `int64` | `int64` | `i64` | `Long` | `scalar Long` | `number` |
| `uint64` | `uint64` | warn | `BigInteger` | `scalar UInt64` | `number` |
| `float32` | `float` | `double` | `Float` | `Float` | `number` |
| `float64` | `double` | `double` | `Double` | `Float` | `number` |
| `bool` | `bool` | `bool` | `Boolean` | `Boolean` | `boolean` |
| `bytes` | `bytes` | `binary` | `Blob` | `scalar Bytes` | `string` |
| `decimal` | `string` | `string` | `BigDecimal` | `scalar Decimal` | `string` |
| `uuid` | `string` | `string` | `String` | `scalar UUID` | `string` |
| `instant` | `google.protobuf.Timestamp` | `string` | `Timestamp` | `scalar DateTime` | `string` |
| `date` | `string` | `string` | `String` | `scalar Date` | `string` |
| `duration` | `google.protobuf.Duration` | `string` | `String` | `scalar Duration` | `string` |

`int` is 64 bits everywhere, and GraphQL needs a custom scalar for it because its `Int` is 32.
Thrift, Smithy, and GraphQL have no unsigned integers, so `uint32` widens to a signed 64-bit type; Thrift has nothing wider for `uint64`.
TypeScript's `number` is a double, so `int`, `int64`, and `uint64` lose precision above 2^53.
`decimal` is a string wherever the target has no exact decimal.
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

- Protobuf has no set, so uniqueness is not enforced.
  A repeated or map field cannot be optional or hold another collection, so those shapes warn.
  A map key must resolve to a string, an integer, or a bool.
  An `edition` directive accepts `2023` or `2024`, reports any other value as an error, and replaces `syntax = "proto3";` with that edition's header; under an edition a `T?` or `T | null` field carries no `optional` label, since every field has explicit presence.
- Smithy names every collection, so the backend synthesizes one shape per distinct collection type, named from its element and key.
  A map key must resolve to a string or a fieldless enum.
- TypeScript types a map key as `string` or `number`, and a fieldless enum key as `Partial<Record<E, V>>`.

A field that is not optional is non-null in GraphQL and `@required` in Smithy.
Thrift fields that are not optional use the default requiredness and never `required`, which Thrift's guidance advises against.

| TDL | protobuf | thrift | smithy | graphql | typescript |
| --- | --- | --- | --- | --- | --- |
| entity, value, mixin | `message` | `struct` | `structure` | `type` | `interface` |
| fieldless enum | `enum` | `enum` | `enum` | `enum` | a union of string literals |
| fielded enum | a `message` with a `oneof` | a `union` | a `union` | a `union` | a discriminated union |
| newtype | expanded | `typedef` | a named simple shape | expanded | `type N = Base` |
| alias | expanded | expanded | expanded | expanded | expanded |

A mixin is emitted too, and a struct including it already carries its fields.

A fielded enum is each target's sum type:

- In protobuf, each variant is a nested message and the enum is a message holding a `oneof` of them.
  A field carrying the `oneof` directive, whose enum's variants each carry one field, is written as a `oneof` of those fields inside its message, and an enum no other field names is then not emitted.
- In Thrift, Smithy, and GraphQL, each variant is a struct named after the enum and the variant, such as `PaymentCard`.
  Smithy targets `Unit` for a variant with no fields.
  A GraphQL object needs a field, so a fieldless variant carries a placeholder `_: Boolean` that is always null.
- In TypeScript, each variant is an interface with a `kind` field holding the variant's name, which a `discriminant` directive renames.

A protobuf enum starts with an `_UNSPECIFIED` value at zero, and its values are prefixed with the enum's name, since enum values share one scope per package.

Protobuf and GraphQL expand a newtype to its base: a protobuf wrapper message would change the wire format, and a GraphQL scalar per newtype would need server code for each.
A TypeScript newtype is a plain alias, so parsed JSON needs no cast.

## Names

A declaration is Pascal case in every target.
A protobuf field is snake case and a protobuf enum value is screaming snake case, per the protobuf style guide.
Every other target writes a field as TDL does.
A `name` directive replaces the name in any target, and a name that collides after conversion, with a keyword, or with a synthesized name warns until `name` resolves it.
A `name` value the target cannot spell as an identifier is a warning of its own.

## Numbering

Protobuf and Thrift field numbers are the wire format, and the IR has none.
A `number(n)` directive pins a field's number, and each unpinned field takes, in declaration order, the lowest number from one that no pin or earlier unpinned field holds and the target does not reserve.
Enum values and variants are numbered the same way.

Two pins on one number are an error, and so is a pin inside a reserved range.
Inserting an unpinned field anywhere but the end renumbers every unpinned field after it, and so does adding a field to a mixin, so a stable schema pins.

Protobuf numbers run to 536870911, with 19000 to 19999 reserved by protobuf.
Thrift numbers run to 32767.

A protobuf message takes a repeatable `reserved` directive of numbers or names, each a `reserved` statement at the top of the message, in the order written.
An unpinned field skips a reserved number, and a field pinned to a reserved number or on a reserved name is refused.
An inlined oneof member is held to both rules, since protobuf counts it as a field of its message.

A protobuf target block takes a repeatable `import(path)` directive, and each path joins every file's imports, which are sorted and written once each.
A protobuf field, message, enum, enum value, service, or rpc takes a repeatable `option(name, value)` directive:

- On a field or an enum value, options are written `name = value` in one bracket list, after any the backend writes itself, such as `deprecated = true`.
- An inlined oneof member carries its variant's options and then its field's, and is deprecated when either is.
- On a message, an enum, or a service, each is an `option name = value;` statement at the top of the body, after any `reserved` statements; on an rpc, in a `{ }` body after its signature.
- An `option("deprecated", ...)` on a deprecated node is dropped, since protoc refuses an option set twice.

## Output

Each backend writes one file per model, since a schema language reads a package as one document.
In protobuf, a `file` directive on a declaration places it in the named file of the package instead, and a file naming a declaration placed in another imports that file by its path.

| Target | File | Namespace |
| --- | --- | --- |
| protobuf | `<package as directories>/<last segment>.proto`, or the `file` directive's name in those directories | `package` directive, else the model's package |
| thrift | `<last segment>.thrift` | `namespace *` from the `package` directive, else the model's package |
| smithy | `<last segment>.smithy` | `namespace` from the `package` directive, else the model's package |
| graphql | `<last segment>.graphql` | none |
| typescript | `<last segment>.ts` | none |

Every file starts with `Code generated by tdl. DO NOT EDIT.` in the target's comment syntax.
Doc comments are carried in each target's form, and a deprecation becomes each target's `deprecated` marker; GraphQL puts a type's deprecation in its description because a type cannot carry `@deprecated`.

## Encodings

TDL defines no wire encoding, so these backends and the Go backend can disagree about one value.
TypeScript types a `duration` and a `date` as strings, while Go's `encoding/json` writes a `time.Duration` as integer nanoseconds and a date as a full timestamp.
A Go pointer marshals to `null`, and TypeScript types `T?` as an absent key.
The `tag` directive is how a Go consumer aligns them; a wire encoding is a decision for the spec.
