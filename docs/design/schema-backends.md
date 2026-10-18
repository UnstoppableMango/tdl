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
- externs and `foreign`, except in protobuf and TypeScript
- entity keys

Protobuf reads `foreign(file, message)` on a declaration or an extern as a message another proto file declares.
The declaration is not emitted, every reference is written as `message`, and each output file referencing it imports `<file>` once.
An extern without `foreign`, whose dependency has a protobuf target block, is the message that dependency generates: it is written `<package>.<Name>` and imported from `<package dirs>/<file>`.
The package is the block's `package` directive or else the dependency's package, the file is its `file` directive or else the package's last segment with `.proto`, and the name is Pascal-cased, so the path matches the dependency's output.
An extern with neither warns.

TypeScript reads `foreign(module, name)` on an extern as a type a hand-written module exports, imported with `import type { name } from "module"`.
An extern without `foreign`, whose dependency has a typescript target block, is the type that dependency generates, named as that block names it.
It is imported from the dependency's file by a path relative to this one, ending in `.js` as Node's module resolution requires; the path comes from both blocks' `out` directives, each relative to the file declaring the block.
Without both `out` directives, or with neither mapping, the extern warns; so does an import whose name another import or a declaration already takes.

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

## Features

What each backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

### Protobuf features

<!-- tdl:features protobuf -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns (`lossy.struct-kind`) | Warns (`lossy.struct-kind`) | a `message` | An entity and a mixin both read back as a value. |
| Mixin `include` | Warns (`lossy.include`) | Warns (`lossy.include`) | the mixin's fields copied into the `message` | Reads back as fields of the message, without the include. |
| Fieldless enum | Yes | Yes | an `enum` with an `_UNSPECIFIED` zero value | |
| Fielded enum | Yes | Yes | a `message` holding a `oneof` of nested messages | |
| Newtype | Warns (`lossy.newtype`) | Warns (`lossy.newtype`) | expanded to its base | A wrapper message would change the wire format, so each use is written as the base. |
| Alias | Warns (`lossy.alias`) | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Warns (`lossy.optional`) | Warns (`lossy.optional`) | an `optional` field | A nullable value is written `optional` too, and reads back as `T?`. |
| List, set, and map | Warns (`lossy.collection`) | Warns (`lossy.collection`) | `repeated` and `map<K, V>` | A set is a `repeated` field, which reads back as a list. |
| `min` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `max` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `length` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `matches` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `oneOf` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `unique` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Protobuf has no constraints, so validation is not generated. |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | Protobuf does not carry ownership, and the referenced message is embedded by value. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | Protobuf has no field defaults. |
| `key` directive | No | No | | Protobuf has no key, and the backend does not declare the directive, so it warns and is ignored. |
| `name` directive | Warns (`lossy.name`) | Warns (`lossy.name`) | the field's written name | The field reads back under the name it is written with, without the directive. |
| `number` directive | Yes | Yes | the field number | Unpinned fields take the lowest free number. |
| Doc comment | Yes | Yes | a leading `//` comment | |
| Type parameters | No | No | | Protobuf has no generics, so `Page` is skipped, and so is every declaration naming it. |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | | The class is not generated, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | Warns (`lossy.unit`) | | The unit is dropped, and `decimal<kg>` is written as a `string`. |
| Import and extern | Yes | No | an `import` of the dependency's file | Reading back is given only this package's files, so the dependency's `.proto` is missing and the import does not compile. |

<!-- /tdl:features -->

### Thrift features

<!-- tdl:features thrift -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns (`lossy.struct-kind`) | Warns (`lossy.struct-kind`) | a `struct` | An entity and a mixin both read back as a value. |
| Mixin `include` | Warns (`lossy.include`) | Warns (`lossy.include`) | the mixin's fields copied into the `struct` | Reads back as fields of the struct, without the include. |
| Fieldless enum | Yes | Yes | an `enum` numbered from 1 | |
| Fielded enum | Yes | Yes | a `union` of one `struct` per variant | |
| Newtype | Yes | Yes | a `typedef` | Read back from the typedef. |
| Alias | Warns (`lossy.alias`) | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Warns (`lossy.optional`) | Warns (`lossy.optional`) | an `optional` field | A nullable value is written `optional` too, and reads back as `T?`. |
| List, set, and map | Yes | Yes | `list<T>`, `set<T>`, and `map<K, V>` | |
| `min` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `max` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `length` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `matches` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `oneOf` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `unique` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Thrift has no constraints, so validation is not generated. |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | Thrift does not carry ownership, and the referenced struct is embedded by value. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | Thrift has field defaults, but the backend does not write them yet. |
| `key` directive | No | No | | Thrift has no key, and the backend does not declare the directive, so it warns and is ignored. |
| `name` directive | Warns (`lossy.name`) | Warns (`lossy.name`) | the field's written name | The field reads back under the name it is written with, without the directive. |
| `number` directive | Yes | Yes | the field id | Unpinned fields take the lowest free id. |
| Doc comment | Yes | Yes | a `/** */` comment | |
| Type parameters | No | No | | Thrift has no generics, so `Page` is skipped, and so is every declaration naming it. |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | | The class is not generated, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | Warns (`lossy.unit`) | | The unit is dropped, and `decimal<kg>` is written as a `string`. |
| Import and extern | No | No | | Externs are not generated yet, and Thrift's `include` is not written, so a declaration naming one is skipped. |

<!-- /tdl:features -->

### Smithy features

<!-- tdl:features smithy -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns (`lossy.struct-kind`) | Warns (`lossy.struct-kind`) | a `structure` | An entity and a mixin both read back as a value. |
| Mixin `include` | Warns (`lossy.include`) | Warns (`lossy.include`) | the mixin's members copied into the `structure` | Reads back as members of the structure, without the include. |
| Fieldless enum | Yes | Yes | an `enum` | |
| Fielded enum | Yes | Yes | a `union` of one `structure` per variant | A variant without fields targets `Unit`. |
| Newtype | Yes | Yes | a named simple, `list`, or `map` shape | |
| Alias | Warns (`lossy.alias`) | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Warns (`lossy.optional`) | Warns (`lossy.optional`) | a member without `@required` | A nullable value and a spelled-out `Option<T>` are written the same way, and read back as `T?`. |
| List, set, and map | Yes | Yes | a named `list` or `map` shape | A set is a `list` with `@uniqueItems`. |
| `min` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | The `@range` trait is not written yet. |
| `max` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | The `@range` trait is not written yet. |
| `length` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | The `@length` trait is not written yet. |
| `matches` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | The `@pattern` trait is not written yet. |
| `oneOf` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Validation is not generated. |
| `unique` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | The `@uniqueItems` trait is written only for a set, not for a `unique` constraint. |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | Smithy does not carry ownership, and the referenced structure is embedded by value. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | The `@default` trait is not written yet. |
| `key` directive | No | No | | The backend does not declare the directive, so it warns and is ignored. |
| `name` directive | Warns (`lossy.name`) | Warns (`lossy.name`) | the member's written name | The member reads back under the name it is written with, without the directive. |
| `number` directive | No | No | | Smithy has no field numbers, and the backend does not declare the directive, so it warns and is ignored. |
| Doc comment | Yes | Yes | a `///` doc comment | |
| Type parameters | No | No | | Smithy has no generics, so `Page` is skipped, and so is every declaration naming it. |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | | The class is not generated, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | Warns (`lossy.unit`) | | The unit is dropped, and `decimal<kg>` is written as a `BigDecimal`. |
| Import and extern | No | No | | Externs are not generated yet, so a declaration naming one is skipped. |

<!-- /tdl:features -->

### GraphQL features

<!-- tdl:features graphql -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns (`lossy.struct-kind`) | Warns (`lossy.struct-kind`) | an object `type` | An entity and a mixin both read back as a value. |
| Mixin `include` | Warns (`lossy.include`) | Warns (`lossy.include`) | the mixin's fields copied into the `type` | Reads back as fields of the type, without the include. |
| Fieldless enum | Yes | Yes | an `enum` | |
| Fielded enum | Yes | Yes | a `union` of one object type per variant | A variant without fields carries a placeholder `_: Boolean`. |
| Newtype | Warns (`lossy.newtype`) | Warns (`lossy.newtype`) | expanded to its base | A custom scalar per newtype would need server code for each, so each use is written as the base. |
| Alias | Warns (`lossy.alias`) | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Warns (`lossy.optional`) | Warns (`lossy.optional`) | a nullable type | A required field is `T!`, and a nullable value reads back as `T?`. |
| List, set, and map | No | No | a list `[T]` | GraphQL has no map type, so a type holding a map is skipped, while lists and sets generate. |
| `min` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `max` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `length` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `matches` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `oneOf` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `unique` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | GraphQL has no constraints, so validation is not generated. |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | GraphQL does not carry ownership. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | An output type's field has no default in GraphQL. |
| `key` directive | No | No | | GraphQL has no key, and the backend does not declare the directive, so it warns and is ignored. |
| `name` directive | Warns (`lossy.name`) | Warns (`lossy.name`) | the field's written name | The field reads back under the name it is written with, without the directive. |
| `number` directive | No | No | | GraphQL has no field numbers, and the backend does not declare the directive, so it warns and is ignored. |
| Doc comment | Yes | Yes | a description | |
| Type parameters | No | No | | GraphQL has no generics, so `Page` is skipped, and so is every declaration naming it. |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | | The class is not generated, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | Warns (`lossy.unit`) | | The unit is dropped, and `decimal<kg>` is written as the `Decimal` scalar. |
| Import and extern | No | No | | Externs are not generated yet, so a declaration naming one is skipped. |

<!-- /tdl:features -->

### TypeScript features

<!-- tdl:features typescript -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns (`lossy.struct-kind`) | Warns (`lossy.struct-kind`) | `interface` | Entities, values, and mixins are all interfaces, so each reads back as a value. |
| Mixin `include` | Warns (`lossy.include`) | Warns (`lossy.include`) | the mixin's properties, copied in | The include is flattened into the fields it copies. |
| Fieldless enum | Yes | Yes | a union of string literals | |
| Fielded enum | Yes | Yes | a union of interfaces discriminated on `kind` | A `discriminant` directive renames `kind`. |
| Newtype | Yes | Yes | `type N = Base` | A plain alias, so parsed JSON needs no cast. |
| Alias | Warns (`lossy.alias`) | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Yes | Yes | `name?: T`, or a union with `null` | |
| List, set, and map | Warns (`lossy.collection`) | Warns (`lossy.collection`) | `T[]` and `Record<K, V>` | A set is an array, and reads back as a list. |
| `min` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Warns unless `min` and `max` together bound at most 16 integers, which are written as a literal union. |
| `max` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | Warns unless `min` and `max` together bound at most 16 integers, which are written as a literal union. |
| `length` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | TypeScript has no type for a length, and validation is not generated. |
| `matches` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | TypeScript has no type for a pattern, and validation is not generated. |
| `oneOf` | Yes | Yes | a union of literal types | |
| `unique` | Warns (`lossy.constraint`) | Warns (`lossy.constraint`) | | A set or a list is an array, which does not enforce uniqueness. |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | TypeScript has no ownership, and a referenced type is embedded by value. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | The default is not written. |
| `key` directive | No | No | | The output is types with no runtime code, so there is no key function to write. |
| `name` directive | Warns (`lossy.name`) | Warns (`lossy.name`) | the property or type name | The new name reads back as the field's own name rather than a directive. |
| `number` directive | No | No | | JSON has no field numbers. |
| Doc comment | Yes | Yes | a JSDoc comment | |
| Type parameters | No | No | | Type parameters are not generated yet, so a parameterized declaration and every one applying it are skipped. |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | | A class is not generated yet, and a conformance to one is dropped. |
| Unit | Warns (`lossy.unit`) | Warns (`lossy.unit`) | | A unit is not generated yet, and a quantity is written as its primitive. |
| Import and extern | Yes | Yes | `import type` | From a `foreign` module, or from the dependency's file when both target blocks set `out`. |

<!-- /tdl:features -->

### JSON Schema features

<!-- tdl:features jsonschema -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | `object` | Entities, values, and mixins are all objects. |
| Mixin `include` | Yes | No | the mixin's properties, copied in | |
| Fieldless enum | Yes | No | `string` with `enum` | |
| Fielded enum | Yes | No | a `oneOf` discriminated on `kind` | A `discriminant` directive renames `kind`. |
| Newtype | Yes | No | a definition | A newtype over another newtype is a `$ref`, which checks the constraints it inherits. |
| Alias | Yes | No | expanded where used | |
| Optional and nullable | Yes | No | left out of `required`, or `anyOf` with `null` | |
| List, set, and map | Yes | No | `array`, `uniqueItems`, and `additionalProperties` | |
| `min` | Yes | No | `minimum` | |
| `max` | Yes | No | `maximum` | |
| `length` | Yes | No | `minLength` and `maxLength` | On a list or set it is `minItems` and `maxItems`, and on a map `minProperties` and `maxProperties`. |
| `matches` | Yes | No | `pattern` | |
| `oneOf` | Yes | No | `enum` | |
| `unique` | Yes | No | `uniqueItems` | |
| `owned` | Yes | No | | Not written: a schema has no ownership, and a referenced entity is a `$ref`. |
| Field default | Yes | No | | Not written, and nothing warns. |
| `key` directive | No | No | | The backend declares no `key` directive, since a schema has no identity. |
| `name` directive | Yes | No | the property or definition name | |
| `number` directive | No | No | | JSON has no field numbers. |
| Doc comment | Yes | No | `description` | |
| Type parameters | No | No | | Type parameters are not generated yet, so a parameterized declaration and every one applying it are skipped. |
| Class | No | No | | A class is not generated yet, so it is skipped with a warning and a conforming type is written without it. |
| Unit | No | No | | A unit is not generated yet, so it is skipped with a warning and a quantity is written as its primitive. |
| Import and extern | No | No | | An extern is not generated yet, so a declaration naming one is skipped. |

<!-- /tdl:features -->

### OpenAPI features

<!-- tdl:features openapi -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | an `object` schema | Entities, values, and mixins are all objects. |
| Mixin `include` | Yes | No | the mixin's properties, copied in | |
| Fieldless enum | Yes | No | `string` with `enum` | |
| Fielded enum | Yes | No | a `oneOf` with a `discriminator` | OpenAPI 2.0 has no `oneOf`, so there it warns and is skipped. |
| Newtype | Yes | No | a schema | |
| Alias | Yes | No | expanded where used | |
| Optional and nullable | Yes | No | left out of `required`, or nullable | Nullable is `anyOf` with `null` in 3.1, `nullable` in 3.0, and `x-nullable` in 2.0. |
| List, set, and map | Yes | No | `array`, `uniqueItems`, and `additionalProperties` | |
| `min` | Yes | No | `minimum` | |
| `max` | Yes | No | `maximum` | |
| `length` | Yes | No | `minLength` and `maxLength` | On a list or set it is `minItems` and `maxItems`, and on a map `minProperties` and `maxProperties`. |
| `matches` | Yes | No | `pattern` | |
| `oneOf` | Yes | No | `enum` | |
| `unique` | Yes | No | `uniqueItems` | |
| `owned` | Yes | No | | Not written: a schema has no ownership, and a referenced entity is a `$ref`. |
| Field default | Yes | No | | Not written, and nothing warns. |
| `key` directive | No | No | | The backend declares no `key` directive, since a schema has no identity. |
| `name` directive | Yes | No | the property or schema name | |
| `number` directive | No | No | | JSON has no field numbers. |
| Doc comment | Yes | No | `description` | |
| Type parameters | No | No | | Type parameters are not generated yet, so a parameterized declaration and every one applying it are skipped. |
| Class | No | No | | A class is not generated yet, so it is skipped with a warning and a conforming type is written without it. |
| Unit | No | No | | A unit is not generated yet, so it is skipped with a warning and a quantity is written as its primitive. |
| Import and extern | No | No | | An extern is not generated yet, so a declaration naming one is skipped. |

<!-- /tdl:features -->

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

### Protobuf

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

### Thrift

`tdl import --from thrift` reads one `.thrift` file, parsed by `cloudwego/thriftgo`; an `include` fails the import.

| Thrift | TDL |
| --- | --- |
| struct | struct |
| union whose every member is a struct nothing else names | enum with fields, one variant per member |
| enum, its values from screaming snake case | enum |
| typedef | newtype |
| `list<T>`, `set<T>`, `map<K, V>`, `optional T` | `[T]`, `{T}`, `{K -> V}`, `T?` |
| `i32`, `i64`, `double`, `binary` | `int32`, `int64`, `float64`, `bytes` |
| `byte`, `i8`, `i16` | `int32`, with `lossy.primitive` |
| exception | struct, with `lossy.unsupported` |
| other union, const, service, `required`, an annotation, a namespace for one language | `lossy.unsupported` |
| a default | `lossy.default` |

A field keeps its name, and a variant's is what its struct's name adds to the union's when the member is that in camel case, and otherwise the member's Pascal case.
The last comment before a node is its doc comment when it is a `/** */` block, and a `Deprecated:` paragraph ending one is the reason of a `deprecated` annotation.
The target block holds a `name` directive where the convention would write another name and a `number` directive on each member allocation would number otherwise.

Generating, every fact the IDL cannot hold warns with its loss code: an `int`, `uint32`, `float32`, `decimal`, `uuid`, `instant`, `date`, or `duration` written as another primitive, collection sugar written by name, `T | null`, an alias expanded, an entity or a mixin, an include, a constraint, `owned`, a default, a name or number that reads back otherwise, the order dependencies impose, a doc comment a `/** */` block cannot hold, a class, a unit, and generics.

Under a `roundtrip` directive, each of those is written instead as an annotation whose key starts `tdl.`:

- on `namespace *`, or `namespace tdl tdl` without a package: `tdl.package` when `package` renames it, `tdl.doc` for the package's doc comment, `tdl.import` for each import, and `tdl.item` for each top-level item with no Thrift form, as its index among the file's declarations and the item as TDL.
- on a definition: `tdl.name`, `tdl.kind` (`entity` or `mixin`), `tdl.conforms`, and `tdl.at`, its index, when the output reorders the declarations.
- on a typedef and a field: `tdl.source`, the newtype or field as TDL where the IDL reads back differently, and on a field `tdl.include`, the mixin whose include copied it.
- on an enum value and a union member: `tdl.name`.
- on any node: `tdl.doc` and `tdl.reason` where its comment does not read back as its doc comment and deprecation.

thriftgo unescapes only `\"` in a literal, so in an annotation's value a backslash before a quote or at the end is written `%5C`, and a percent sign `%25`.
The normal form for comparing schemas is the thriftgo AST as JSON, without comments, each kind of definition sorted by name.

### GraphQL

`tdl import --from graphql` reads one schema, parsed and validated by `vektah/gqlparser/v2`.
The file's name is the package, except `schema`, the name of a model without one.

| GraphQL | TDL |
| --- | --- |
| object type | struct |
| union whose every member is an object type nothing else names | enum with fields, one variant per member |
| object type holding only a nullable `_: Boolean`, as a union member | variant without fields |
| enum, its values from screaming snake case | enum |
| `T!`, `T`, `[T]` | `T`, `T?`, `[T?]?` |
| `String`, `Int`, `Float`, `Boolean` | `string`, `int32`, `float64`, `bool` |
| `Long`, `UInt64`, `Bytes`, `Decimal`, `UUID`, `DateTime`, `Date`, `Duration` | `int64`, `uint64`, `bytes`, `decimal`, `uuid`, `instant`, `date`, `duration` |
| `ID` | `string`, with `lossy.primitive` |
| interface, input type, another scalar, other union, type extension, a field's arguments, the root operation types, a directive | `lossy.unsupported` |

A field keeps its name, and a variant's is what its object type's name adds to the union's, or else the type's own.
A description is a doc comment, and a type's ending in a `Deprecated:` paragraph, or `Deprecated.`, is its deprecation; `@deprecated` is a field's or an enum value's.
The target block holds a `name` directive where the convention would write another name.

Generating, every fact the schema cannot hold warns with its loss code: an `int`, `uint32`, or `float32` written as another primitive, a set, collection sugar written by name, `T | null`, a newtype or alias expanded, an entity or a mixin, an include, a constraint, `owned`, a default, a name that reads back otherwise, a package the file's name does not hold, a doc comment a description cannot hold, a class, a unit, and generics.

Under a `roundtrip` directive, each of those is written instead as a `@tdl` directive, which the schema defines, with one argument per use:

- on `extend schema`: `package` when the file's name does not hold it, `doc` for the package's doc comment, `import` for each import, and `item` for each top-level item with no GraphQL type, as its index among the file's declarations and the item as TDL. A newtype, an alias, a class, a unit, an instance, a target block, and a declaration GraphQL cannot express are each one.
- on a type: `name`, `kind` (`entity` or `mixin`), and `conforms`.
- on a field: `source`, the field as TDL where the schema reads back differently, and `include`, the mixin whose include copied it.
- on an enum value and a variant's object type: `name`.
- on any node: `doc`, and `reason` when it is deprecated, where its description does not read back as its doc comment and deprecation.

The normal form for comparing schemas is gqlparser's schema document, formatted without comments, its definitions sorted by name and its scalars without descriptions, since no model carries a scalar's.

### TypeScript

`tdl import --from typescript` reads one `.ts` file, parsed by the TypeScript compiler, which an embedded script runs under `node`; both `node` and `tsc` must be on `PATH`.
The file's name is the package, except `model`, the name of a model without one.

| TypeScript | TDL |
| --- | --- |
| interface | struct |
| union of interfaces nothing else names, each tagged by one string-literal property | enum with fields, one variant per member, discriminated on that property |
| union of string literals, each a TDL name | enum |
| `never` | enum without variants |
| any other type alias | newtype |
| `p?: T`, `p: T \| null`, `p?: T \| null` | `T?`, `T \| null`, `T? \| null` |
| `T \| null` in an element or a value | `T?` |
| `T[]`, `Array<T>`, `ReadonlyArray<T>` | `[T]` |
| `Record<K, V>`, `Partial<Record<K, V>>` | `{K -> V}` |
| `string`, `number`, `boolean` | `string`, `float64`, `bool` |
| a union of string or number literals | `string` or `float64`, `where { oneOf(...) }` |
| TypeScript `enum` | enum, with `lossy.unsupported` |
| a class, a method, an index signature, an object literal type, another union or type, `extends`, `readonly` | `lossy.unsupported` |
| a declaration not exported | the declaration, with `lossy.unsupported` |

A property keeps its name, unless it is not a TDL identifier: then it reads in camel case with a `name` directive.
A variant's name is what its interface's name adds to the union's, or else the interface's own, with a `name` directive.
The last `/** */` comment is a doc comment: its lines before the first tag are the doc, and `@deprecated` the deprecation; any other tag warns `lossy.doc`.
The target block holds `name` and `discriminant` directives where the convention would write another.

Generating, every fact the file cannot hold warns with its loss code: a primitive written as `string` or `number` that reads back as another, a set, collection sugar written by name, an option or nullable chain the file cannot write, a newtype or alias expanded, an entity or a mixin, an include, a constraint, `owned`, a default, a name that reads back otherwise, a package the file's name does not hold, a doc comment JSDoc cannot hold, a class, a unit, and generics.

Under a `roundtrip` directive, each is written instead as a JSDoc `@tdl <key> <value>` tag, its value a Go string literal with `*/` written `*\u002f`:

- `source` on a declaration that loses anything: the declaration as TDL.
- on an `export {};` at the end of the file: `package` when the file's name does not hold it, `doc` for the package's doc comment, `import` for each import, and `item` for each top-level item with no TypeScript form, as its index among the file's declarations and the item as TDL.

The normal form for comparing files is the compiler's outline as JSON, without positions, each JSDoc comment read into its doc, deprecation, and tags, without whether a property's name is quoted, and with `Array<T>` written `T[]`.

### Smithy

`tdl import --from smithy` reads `.smithy` files of one namespace, which the Smithy CLI's `smithy ast` turns into its JSON AST; `smithy` must be on `PATH`.
The namespace is the package.

| Smithy | TDL |
| --- | --- |
| structure | struct |
| union whose every member targets `Unit` or a structure nothing else names | enum with fields, one variant per member |
| enum, each member's value its variant's name | enum |
| a list or map named for what it holds, with no trait but `@sparse` and `@uniqueItems` | the collection a member holds |
| any other list or map, and a simple shape | newtype |
| a member without `@required` | `T?` |
| `@sparse`, `@uniqueItems` | an element `T?`, `{T}` |
| `String`, `Integer`, `Long`, `BigInteger`, `Float`, `Double` | `string`, `int32`, `int64`, `uint64`, `float32`, `float64` |
| `Boolean`, `Blob`, `BigDecimal`, `Timestamp` | `bool`, `bytes`, `decimal`, `instant` |
| `Short`, `Byte` | `int32`, with `lossy.primitive` |
| intEnum | enum, with `lossy.unsupported` |
| `Document`, service, operation, resource, mixins, other union, metadata, any other trait | `lossy.unsupported` |

A shape keeps its name, a member its own, a union member's variant is its Pascal case, and an enum member's variant is its value when that is a TDL name, else its Pascal case with `lossy.name`.
A doc comment is a doc comment, and `@deprecated` with its `message` a deprecation; a variant takes its union member's, else its structure's.
The JSON AST has no positions, so declarations are ordered, and diagnostics placed, by the line each shape's statement starts on.
The target block holds a `name` directive where the convention would write another name: a shape's Pascal case, an enum member's screaming snake case, and a variant structure's union and variant names joined.

Generating, every fact the model cannot hold warns with its loss code: an `int`, `uint32`, `uuid`, `date`, or `duration` written as another primitive, collection sugar written by name, `T | null` or `Option<T>`, a newtype or alias expanded, an entity or a mixin, an include, a constraint, `owned`, a default, a name that reads back otherwise, a package `package` renames, the package's doc comment, a doc comment's trailing whitespace, a class, a unit, and generics.

Under a `roundtrip` directive, what the model cannot hold is written instead:

- `@tdl#source` on a declaration whose shapes do not read back as it: the declaration as TDL. Generating reads its own shapes back to decide, so a declaration that reads back whole carries nothing. The trait is defined in a generated `tdl.smithy`, written only when a shape uses it.
- `metadata tdl`: `package` when the namespace does not hold it, `doc` for the package's doc comment, `import` for each import, and `item` for each top-level item with no shape, as its index among the file's declarations and the item as TDL. It is always written, since it is how import knows the files are annotated and infers no directive.

The normal form for comparing models is the `smithy ast` JSON of the shapes each file declares, and of the metadata in the file that writes it.

## Encodings

TDL defines no wire encoding, so these backends and the Go backend can disagree about one value.
TypeScript types a `duration` and a `date` as strings, while Go's `encoding/json` writes a `time.Duration` as integer nanoseconds and a date as a full timestamp.
A Go pointer marshals to `null`, and TypeScript types `T?` as an absent key.
The `tag` directive is how a Go consumer aligns them; a wire encoding is a decision for the spec.
