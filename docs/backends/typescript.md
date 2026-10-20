# TypeScript

The `typescript` backend writes one `.ts` file per model of JSON wire types: interfaces, both enum shapes, and newtypes as aliases, with no runtime code.
[schema-backends.md](../design/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
