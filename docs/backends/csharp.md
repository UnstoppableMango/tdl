# C\#

The `csharp` backend writes records, enums, and newtypes as readonly record structs, one `.g.cs` file per declaration or one for the whole target.
It is a .NET plugin, `tdl-gen-csharp`, which `tdl gen` runs from `PATH`.
[csharp-backend.md](../architecture/csharp-backend.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

<!-- tdl:features csharp -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a `sealed record` | Entities, values, and mixins are all records. |
| Mixin `include` | Yes | No | the mixin's properties, copied in | |
| Fieldless enum | Yes | No | a C# `enum` | |
| Fielded enum | Yes | No | an `abstract record` with a nested `sealed record` per variant | A private constructor closes it to its variants. |
| Newtype | Yes | No | a `readonly record struct` | |
| Alias | Yes | No | expanded where used | |
| Optional and nullable | Yes | No | a nullable property | An optional `T?` may be left out, while a union with `null` is `required` and may be set to null. |
| List, set, and map | Yes | No | `IImmutableList<T>`, `IImmutableSet<T>`, and `IImmutableDictionary<K, V>` | A record holding one writes its own `Equals` and `GetHashCode`, since .NET compares collections by reference. |
| `min` | Planned (phase 4) | No | | Validation is not generated yet. |
| `max` | Planned (phase 4) | No | | Validation is not generated yet. |
| `length` | Planned (phase 4) | No | | Validation is not generated yet. |
| `matches` | Planned (phase 4) | No | | Validation is not generated yet. |
| `oneOf` | Planned (phase 4) | No | | Validation is not generated yet. |
| `unique` | Planned (phase 4) | No | | Validation is not generated yet. |
| `owned` | Yes | No | | C# has no ownership, so it is not written, and nothing warns. |
| Field default | Yes | No | a property initializer | The property stops being `required`. |
| `key` directive | Yes | No | a `Key()` method | A key of several fields returns a `<Name>Key` record struct. |
| `name` directive | Yes | No | the identifier, as written | |
| `number` directive | No | No | | C# has no field numbers, and the backend does not declare the directive, so it warns without a loss code and is ignored. |
| Doc comment | Yes | No | a `/// <summary>` comment | |
| Type parameters | Planned (phase 3) | No | | `Page` is skipped with a warning, and `Carts`, which applies it, with a warning that carries no loss code. |
| Class | Planned (phase 3) | No | | The class is not generated yet, and a conformance to it is dropped. |
| Unit | Warns (`lossy.unit`) | No | | C# has no units of measure, so `decimal<kg>` is written as `decimal`. |
| Import and extern | Yes | No | the dependency's type, `global::`-qualified | Its namespace is the dependency block's `namespace` directive, or else the one its package derives. |

<!-- /tdl:features -->
