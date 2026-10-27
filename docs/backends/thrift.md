# Thrift

The `thrift` backend writes one `.thrift` file per model: structs, both enum shapes, newtypes as typedefs, collections, and `number` pins.
[schema-backends.md](../architecture/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
