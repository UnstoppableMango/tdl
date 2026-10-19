# GraphQL

The `graphql` backend writes one schema per model of output types: both enum shapes, custom scalars, and lists, but no maps.
[schema-backends.md](../design/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
