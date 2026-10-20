# Haskell

The `haskell` backend writes one module per model: records, sum types, newtypes, type synonyms, key functions, type parameters, and classes as type classes.
[haskell-backend.md](../design/haskell-backend.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

<!-- tdl:features haskell -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a `data` record | Entities, values, and mixins are all records. |
| Mixin `include` | Yes | No | the mixin's fields, copied in | |
| Fieldless enum | Yes | No | a sum type deriving `Enum` and `Bounded` | |
| Fielded enum | Yes | No | a sum type with record constructors | |
| Newtype | Yes | No | a `newtype` | |
| Alias | Yes | No | a `type` synonym | |
| Optional and nullable | Yes | No | `Maybe T` | An optional and a nullable value are the same type, since Haskell has no wire to tell them apart on. |
| List, set, and map | Yes | No | `[T]`, `Set T`, and `Map K V` | |
| `min` | Planned (phase 5) | No | | Validation is not generated yet. |
| `max` | Planned (phase 5) | No | | Validation is not generated yet. |
| `length` | Planned (phase 5) | No | | Validation is not generated yet. |
| `matches` | Warns (`lossy.constraint`) | No | | `base` has no regular expressions, so `matches` warns, and phase 5 keeps it that way. |
| `oneOf` | Planned (phase 5) | No | | Validation is not generated yet. |
| `unique` | Planned (phase 5) | No | | Validation is not generated yet. |
| `owned` | Yes | No | | Haskell has no ownership, so it is not written, and nothing warns. |
| Field default | Yes | No | | Not written, and nothing warns. |
| `key` directive | Yes | No | a key function, such as `cartKey` | A key of several fields returns a key record. |
| `name` directive | Yes | No | the Haskell identifier | |
| `number` directive | No | No | | Haskell has no field numbers, and the backend does not declare the directive, so it warns without a loss code and is ignored. |
| Doc comment | Yes | No | a Haddock comment | |
| Type parameters | Yes | No | type variables | `Functor`, `Foldable`, and `Traversable` are derived where they apply. |
| Class | Yes | No | a type class with a `HasField` superclass per field | Each conformance is an instance. |
| Unit | Planned (phase 6) | No | | `decimal<kg>` is refused until units are generated, so `Parcel` is skipped with a warning that carries no loss code. |
| Import and extern | Yes | No | an import of the dependency's module | The module is the dependency block's `module` directive, or else its package. |

<!-- /tdl:features -->
