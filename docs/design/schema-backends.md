# Schema backends

Design document.

Seven backends, each in `backend/<name>`, turn a resolved model into a schema language: `protobuf`, `thrift`, `smithy`, `graphql`, `typescript`, `jsonschema`, and `openapi`.
The last three describe JSON on the wire.

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

`backend/internal/jsonschema` builds JSON Schema definitions for `jsonschema` and `openapi`, with a `Dialect` stating what differs between the documents they write ([OpenAPI](#openapi) has the table).

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

Constraints warn and the declaration is still emitted, except in JSON Schema, which has a keyword for each standard one, and where TypeScript has an exact literal type (see [Constraints](#constraints)).
`Field.default_value` and `owned` are not read, and a referenced entity is embedded by value.

GraphQL emits output types only; input types would be a second copy of every type with different union rules.

## The type mapping

**warn** below means a positioned warning, with the reaching declaration skipped.

| TDL | protobuf | thrift | smithy | graphql | typescript | jsonschema |
| --- | --- | --- | --- | --- | --- | --- |
| `string` | `string` | `string` | `String` | `String` | `string` | `string` |
| `int` | `int64` | `i64` | `Long` | `scalar Long` | `number` | `integer` |
| `int32` | `int32` | `i32` | `Integer` | `Int` | `number` | `integer`, 32-bit bounds |
| `uint32` | `uint32` | `i64` | `Long` | `scalar Long` | `number` | `integer`, 32-bit unsigned bounds |
| `int64` | `int64` | `i64` | `Long` | `scalar Long` | `number` | `integer` |
| `uint64` | `uint64` | warn | `BigInteger` | `scalar UInt64` | `number` | `integer`, `minimum: 0` |
| `float32` | `float` | `double` | `Float` | `Float` | `number` | `number` |
| `float64` | `double` | `double` | `Double` | `Float` | `number` | `number` |
| `bool` | `bool` | `bool` | `Boolean` | `Boolean` | `boolean` | `boolean` |
| `bytes` | `bytes` | `binary` | `Blob` | `scalar Bytes` | `string` | `string`, `contentEncoding: base64` |
| `decimal` | `string` | `string` | `BigDecimal` | `scalar Decimal` | `string` | `string` |
| `uuid` | `string` | `string` | `String` | `scalar UUID` | `string` | `string`, `format: uuid` |
| `instant` | `google.protobuf.Timestamp` | `string` | `Timestamp` | `scalar DateTime` | `string` | `string`, `format: date-time` |
| `date` | `string` | `string` | `String` | `scalar Date` | `string` | `string`, `format: date` |
| `duration` | `google.protobuf.Duration` | `string` | `String` | `scalar Duration` | `string` | `string`, `format: duration` |

`int` is 64 bits everywhere, and GraphQL needs a custom scalar for it because its `Int` is 32.
Thrift, Smithy, and GraphQL have no unsigned integers, so `uint32` widens to a signed 64-bit type; Thrift has nothing wider for `uint64`.
TypeScript's `number` is a double, so `int`, `int64`, and `uint64` lose precision above 2^53.
`decimal` is a string wherever the target has no exact decimal.
A GraphQL custom scalar is declared only when something uses it.

| TDL | protobuf | thrift | smithy | graphql | typescript | jsonschema |
| --- | --- | --- | --- | --- | --- | --- |
| `List<T>` | `repeated T` | `list<T>` | a `list` shape | `[T!]` | `T[]` | `array` |
| `Set<T>` | `repeated T` | `set<T>` | a `@uniqueItems` `list` shape | `[T!]` | `T[]` | `array`, `uniqueItems` |
| `Map<K, V>` | `map<K, V>` | `map<K, V>` | a `map` shape | warn | `Record<K, V>` | `object`, `additionalProperties: V` |
| `T?` field | `optional` | `optional` | no `@required` | `T` | `name?: T` | not `required` |
| `T \| null` field | `optional` | `optional` | no `@required` | `T` | `name: T \| null` | `required`, `anyOf` with `null` |
| `[T?]` | warn | warn | `@sparse` | `[T]` | `(T \| null)[]` | `items` `anyOf` with `null` |
| `T? \| null` | warn | warn | warn | warn | `name?: T \| null` | not `required`, `anyOf` with `null` |

- Protobuf has no set, so uniqueness is not enforced.
  A repeated or map field cannot be optional or hold another collection, so those shapes warn.
  A map key must resolve to a string, an integer, or a bool.
  An `edition` directive accepts `2023` or `2024`, reports any other value as an error, and replaces `syntax = "proto3";` with that edition's header; under an edition a `T?` or `T | null` field carries no `optional` label, since every field has explicit presence.
- Smithy names every collection, so the backend synthesizes one shape per distinct collection type, named from its element and key.
  A map key must resolve to a string or a fieldless enum.
- TypeScript types a map key as `string` or `number`, and a fieldless enum key as `Partial<Record<E, V>>`.
- JSON Schema checks a map key with `propertyNames`: a newtype over a string is a `$ref` to it, a string is unchecked, an integer key is a pattern of decimal digits, and a fieldless enum key is a `$ref` to the enum.
  Any other key warns.

A field that is not optional is non-null in GraphQL and `@required` in Smithy.
Thrift fields that are not optional use the default requiredness and never `required`, which Thrift's guidance advises against.

| TDL | protobuf | thrift | smithy | graphql | typescript | jsonschema |
| --- | --- | --- | --- | --- | --- | --- |
| entity, value, mixin | `message` | `struct` | `structure` | `type` | `interface` | `object` |
| fieldless enum | `enum` | `enum` | `enum` | `enum` | a union of string literals | `string` with `enum` |
| fielded enum | a `message` with a `oneof` | a `union` | a `union` | a `union` | a discriminated union | a discriminated `oneOf` |
| newtype | expanded | `typedef` | a named simple shape | expanded | `type N = Base` | a definition |
| alias | expanded | expanded | expanded | expanded | expanded | expanded |

A mixin is emitted too, and a struct including it already carries its fields.

A fielded enum is each target's sum type:

- In protobuf, each variant is a nested message and the enum is a message holding a `oneof` of them.
  A field carrying the `oneof` directive, whose enum's variants each carry one field, is written as a `oneof` of those fields inside its message, and an enum no other field names is then not emitted.
- In Thrift, Smithy, and GraphQL, each variant is a struct named after the enum and the variant, such as `PaymentCard`.
  Smithy targets `Unit` for a variant with no fields.
  A GraphQL object needs a field, so a fieldless variant carries a placeholder `_: Boolean` that is always null.
- In TypeScript, each variant is an interface with a `kind` field holding the variant's name, which a `discriminant` directive renames.
- In JSON Schema, the enum is a `oneOf` of one object per variant, titled with the variant's name, and the discriminant is a required property holding a `const` of that name, renamed the same way.

A protobuf enum starts with an `_UNSPECIFIED` value at zero, and its values are prefixed with the enum's name, since enum values share one scope per package.

Protobuf and GraphQL expand a newtype to its base: a protobuf wrapper message would change the wire format, and a GraphQL scalar per newtype would need server code for each.
A TypeScript newtype is a plain alias, so parsed JSON needs no cast.
A JSON Schema newtype is a definition holding its base and the constraints it writes itself; a base that is another newtype is a `$ref`, which checks the constraints inherited from it.

A JSON Schema object is open: it accepts properties it does not declare, so a newer producer adding a field still validates.
A `closed` directive on a struct or an enum, or on the target block for every one, writes `additionalProperties: false`.

## Constraints

JSON Schema writes each standard constraint as the keyword that checks it, chosen by what the constrained type holds once newtypes are expanded.

| Constraint | number | string | list or set | map |
| --- | --- | --- | --- | --- |
| `min(n)`, `max(n)` | `minimum`, `maximum` | warn | warn | warn |
| `length(n)`, `length(a..b)` | warn | `minLength`, `maxLength` | `minItems`, `maxItems` | `minProperties`, `maxProperties` |
| `matches(/re/)` | warn | `pattern` | warn | warn |
| `oneOf(...)` | `enum` | `enum` | warn | warn |
| `unique` | warn | warn | `uniqueItems` | warn |

`oneOf` also applies to a fieldless enum, where it narrows the variants a field accepts.
Here **warn** leaves the declaration emitted without the keyword, as a constraint warns in every other target, and so does a constraint name the spec does not define.
A constraint on a `T?` or `T | null` field applies to the value when one is present.
`matches` copies the pattern as written, and JSON Schema reads it as an ECMA-262 regular expression.

TypeScript writes a constraint as a literal union where one is exact, and warns for the rest:

- `oneOf` on a string, a number, or a fieldless enum is the union of its values: `"open" | "closed"`, `1 | 2`.
- An integer bounded by both `min` and `max`, with at most 16 values between them, is the union of every one: `min(1) max(6)` is `1 | 2 | 3 | 4 | 5 | 6`.
  A `min` and `max` beside a `oneOf` are enforced when every value lies between them.

A field's own constraints combine with those of the newtype it names, so `tinier: Small where { max(2) }` over `type Small: int where { min(1) max(3) }` is `1 | 2`, and a field with none of its own keeps the newtype's name.

## Names

A declaration is Pascal case in every target.
A protobuf field is snake case and a protobuf enum value is screaming snake case, per the protobuf style guide.
Every other target writes a field as TDL does.
A `name` directive replaces the name in any target, and a name that collides after conversion, with a keyword, or with a synthesized name warns until `name` resolves it.
A `name` value the target cannot spell as an identifier is a warning of its own.
A JSON Schema definition name is held to letters, digits, `_`, `.`, and `-`, since it is written into a URI fragment unescaped.

## Numbering

Protobuf and Thrift field numbers are the wire format, and the IR has none.
A `number(n)` directive pins a field's number, and each unpinned field takes, in declaration order, the lowest number from one that no pin or earlier unpinned field holds and the target does not reserve.
Enum values and variants are numbered the same way.

Two pins on one number are an error, and so is a pin inside a reserved range.
Inserting an unpinned field anywhere but the end renumbers every unpinned field after it, and so does adding a field to a mixin, so a stable schema pins.

Protobuf numbers run to 536870911, with 19000 to 19999 reserved by protobuf.
Thrift numbers run to 32767.

A protobuf message takes a repeatable `reserved` directive of numbers or names, each a `reserved` statement at the top of the message, in the order written.
An unpinned field skips a reserved number, and a field pinned to a reserved number or on a reserved name is an error.
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
| jsonschema | `<last segment>.schema.json` | none |
| openapi | `<last segment>.openapi.yaml`, or `.openapi.json` under `format("json")` | none |

Every file starts with `Code generated by tdl. DO NOT EDIT.` in the target's comment syntax.
JSON has no comments, so a JSON Schema document carries it in `$comment`.

A JSON Schema document holds every declaration under `$defs`, and validates nothing itself unless the target block's `root` directive names the declaration it does, which becomes a top-level `$ref`.
An `id` directive sets the document's `$id`.
A `draft` directive picks the dialect, `2020-12` by default or `draft-07`, and reports any other value as an error.
Under `draft-07` the definitions are under `definitions`, a `$ref` beside other keywords moves into an `allOf`, since that draft ignores a `$ref`'s siblings, and a deprecation is written into the description, since that draft has no `deprecated` keyword.
Doc comments are carried in each target's form, and a deprecation becomes each target's `deprecated` marker; GraphQL puts a type's deprecation in its description because a type cannot carry `@deprecated`.

## OpenAPI

An OpenAPI document holds the model's schemas and no operations: `paths` is empty, since TDL has no HTTP method or route to write one from.
Every declaration is a schema under `components.schemas`, or `definitions` in 2.0, mapped as JSON Schema maps it, constraints included, in the dialect the version speaks.

An `openapi` directive on the target block picks the version, `3.1` by default, `3.0`, or `2.0`, and reports any other value as an error.
A `format` directive picks `yaml`, the default, or `json`.
`title` and `version` directives set `info.title` and `info.version`, which default to the model's package and `0.0.0`.
A YAML document carries the generated-code line as a comment, and a JSON one carries it in an `x-generated` extension.

| | 3.1 | 3.0 | 2.0 |
| --- | --- | --- | --- |
| `openapi` or `swagger` | `3.1.1` | `3.0.4` | `2.0` |
| schemas under | `components.schemas` | `components.schemas` | `definitions` |
| `T \| null` | `anyOf` with `null` | `nullable: true` | `x-nullable: true` |
| a `$ref` beside other keywords | kept | moved into an `allOf` | moved into an `allOf` |
| deprecation | `deprecated: true` | `deprecated: true` | written into the description |
| discriminant value | `const` | one-member `enum` | none |
| `bytes` | `contentEncoding: base64` | `format: byte` | `format: byte` |
| map key schema | `propertyNames` | none | none |
| fielded enum | `oneOf` with a `discriminator` | `oneOf` with a `discriminator` | warn |

A schema of every version carries OpenAPI's width formats: `int32`, `int64` for `int`, `int64`, and `uint32`, `float`, and `double`.
An `int32` writes no bounds beside its format, which already says them.

A fielded enum is a `oneOf` of one schema per variant, each its own component named as TypeScript names the variant's interface, such as `PaymentCard`, and a `name` directive on the variant renames it.
The `discriminator` names the discriminant property and maps each variant's name to its schema, which is the shape OpenAPI code generators read.
OpenAPI 2.0 has no `oneOf`, so there a fielded enum warns and is skipped, along with every declaration naming it.

## Import

`tdl import --from protobuf` reads `.proto` files of one package, compiled together by `bufbuild/protocompile`, and [reverse.md](reverse.md) describes the rest of the direction.

| protobuf | TDL |
| --- | --- |
| message | struct |
| message holding only a oneof named `variant` of its own nested messages, each member named for its message | enum with fields |
| enum, its zero value dropped and its values unprefixed | enum |
| other oneof | a field of a made-up enum, one variant per member, with a `oneof` directive |
| `repeated T`, `map<K, V>`, `optional T` | `[T]`, `{K -> V}`, `T?` |
| `google.protobuf.Timestamp`, `google.protobuf.Duration` | `instant`, `duration` |
| `sint`, `fixed`, and `sfixed` integers | the integer of the same width, with `lossy.primitive` |
| a wrapper such as `google.protobuf.StringValue` | `T?`, with `lossy.optional` |
| another file's message | an empty struct with a `foreign` directive |
| nested message or enum | hoisted to the top level, with `lossy.unsupported` |
| service | `lossy.unsupported` |

A field's name is its camel case, an enum value's is its Pascal case after the enum's prefix, and a message keeps its own.
A leading comment is a doc comment, and a `Deprecated:` line ending one is the reason of a `deprecated` option.

The target block holds what regeneration reads: a `name` directive where the convention would write another name, a `number` directive on each member allocation would number otherwise, `reserved`, `edition`, `file`, `option` for each option but `deprecated`, and `import` for an import nothing uses.

Generating, every fact the schema cannot hold warns with its loss code: an `int`, `uuid`, `decimal`, or `date` written as another primitive, a set, a newtype or alias expanded, `T | null`, a `T?` with presence it already had, an entity or a mixin, an include, a constraint, `owned`, a default, a name or number that reads back otherwise, a class, a unit, and generics.

Under a `roundtrip` directive, each of those is written instead as a custom option declared in `tdl/annotations.proto`, which the output includes:

- `(tdl.file)` on the first file: the TDL package when `package` renames it, the imports, and every top-level item with no protobuf form, as TDL with its index among the file's declarations. A newtype, an alias, a class, a unit, an instance, a target block, and a declaration protobuf cannot express are each one.
- `(tdl.message)` and `(tdl.enum)`: the TDL name, the kind (`KIND_ENTITY` or `KIND_MIXIN`), the conformance list beyond `Entity`, and an index when the output spans files.
- `(tdl.field)` and `(tdl.oneof)`: the field as TDL where the schema reads back differently, and the mixin whose include copied it.
- `(tdl.value)`: an enum variant's name.
- `(tdl.service)`: the service's declaration as TDL, since a service is not read yet.

Reading annotated files, the reader takes the annotations and infers no directive, so the target blocks come back from `(tdl.file)` as they were written.
The normal form for comparing schemas is each file's `FileDescriptorProto` without source info, its imports sorted and reserved ranges merged.

## Encodings

TDL defines no wire encoding, so these backends and the Go backend can disagree about one value.
TypeScript types a `duration` and a `date` as strings, while Go's `encoding/json` writes a `time.Duration` as integer nanoseconds and a date as a full timestamp.
A Go pointer marshals to `null`, and TypeScript types `T?` as an absent key.
The `tag` directive is how a Go consumer aligns them; a wire encoding is a decision for the spec.
