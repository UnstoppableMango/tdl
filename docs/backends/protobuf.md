# Protocol Buffers

The `protobuf` backend writes `.proto` files: messages, both enum shapes, newtypes expanded to their base, collections, `number` pins, and services.
[schema-backends.md](../architecture/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back, and counts only what reading back loses, so a loss Generate already shows is not counted again.

<!-- tdl:features protobuf -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | Warns (`lossy.struct-kind`) | a `message` | An entity and a mixin both read back as a value. |
| Mixin `include` | Yes | Warns (`lossy.include`) | the mixin's fields copied into the `message` | Reads back as fields of the message, without the include. |
| Fieldless enum | Yes | Yes | an `enum` with an `_UNSPECIFIED` zero value | |
| Fielded enum | Yes | Yes | a `message` holding a `oneof` of nested messages | |
| Newtype | Warns (`lossy.newtype`) | Yes | expanded to its base | A wrapper message would change the wire format, so each use is written as the base. |
| Alias | Yes | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Yes | Warns (`lossy.optional`) | an `optional` field | A nullable value is written `optional` too, and reads back as `T?`. |
| List, set, and map | Warns (`lossy.collection`) | Yes | `repeated` and `map<K, V>` | A set is a `repeated` field, which reads back as a list. |
| `min` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `max` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `length` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `matches` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `oneOf` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `unique` | Warns (`lossy.constraint`) | Yes | | Protobuf has no constraints, so validation is not generated. |
| `owned` | Warns (`lossy.owned`) | Yes | | Protobuf does not carry ownership, and the referenced message is embedded by value. |
| Field default | Warns (`lossy.default`) | Yes | | Protobuf has no field defaults. |
| `key` directive | No | No | | Protobuf has no key, and the backend does not declare the directive, so it warns and is ignored. |
| `name` directive | Yes | Warns (`lossy.name`) | the field's written name | The field reads back under the name it is written with, without the directive. |
| `number` directive | Yes | Yes | the field number | Unpinned fields take the lowest free number. |
| Doc comment | Yes | Yes | a leading `//` comment | |
| Type parameters | No | No | | Protobuf has no generics, so `Page` is skipped, and so is every declaration naming it. |
| Class | Warns (`lossy.class`) | Yes | | The class is not generated, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | Yes | | The unit is dropped, and `decimal<kg>` is written as a `string`. |
| Import and extern | Yes | No | an `import` of the dependency's file | Reading back is given only this package's files, so the dependency's `.proto` is missing and the import does not compile. |

<!-- /tdl:features -->
