# Smithy

The `smithy` backend writes one Smithy IDL 2.0 file per model: structures, both enum shapes, and named collection shapes.
[schema-backends.md](../architecture/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
