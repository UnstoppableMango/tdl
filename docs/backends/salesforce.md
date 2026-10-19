# Salesforce

The `salesforce` backend writes Salesforce DX source: a custom object per entity, and Apex for values and enums.
[salesforce-backend.md](../design/salesforce-backend.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

<!-- tdl:features salesforce -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a `CustomObject`, or an Apex class | An entity is a custom object; a value and a mixin are Apex classes. |
| Mixin `include` | Yes | No | the mixin's fields, copied in | |
| Fieldless enum | Yes | No | an Apex `enum`, and a restricted `Picklist` on an object | |
| Fielded enum | Yes | No | an Apex class with a `Kind` enum and an inner class per variant | |
| Newtype | Yes | No | expanded to its base | |
| Alias | Yes | No | expanded to its base | |
| Optional and nullable | Yes | No | a field that is not required | Every Apex variable may be null, so in Apex an optional type is the type it wraps. |
| List, set, and map | Yes | No | Apex `List<T>`, `Set<T>`, and `Map<K, V>` | On an object only a set of a fieldless enum has a column, a `MultiselectPicklist`. |
| `min` | Warns (`lossy.constraint`) | No | | Validation is not generated yet. |
| `max` | Warns (`lossy.constraint`) | No | | Validation is not generated yet. |
| `length` | Warns (`lossy.constraint`) | No | | Validation is not generated yet; a `length` directive sets a `Text` field's length. |
| `matches` | Warns (`lossy.constraint`) | No | | Validation is not generated yet. |
| `oneOf` | Warns (`lossy.constraint`) | No | | Validation is not generated yet. |
| `unique` | Warns (`lossy.constraint`) | No | | Validation is not generated yet. |
| `owned` | No | No | | `owned` is not read, and the corpus's owned list has no field type, so the object is written without it. |
| Field default | Yes | No | | Not written, and nothing warns. |
| `key` directive | Yes | No | a unique external ID | One `Text` or `Number` field; a key of several fields warns. |
| `name` directive | Yes | No | the API name, or the Apex name | |
| `number` directive | No | No | | The backend declares no `number` directive, since an org has no field numbers. |
| Doc comment | Yes | No | `description`, or ApexDoc | |
| Type parameters | No | No | | Type parameters are not generated yet, so a parameterized declaration and every one applying it are skipped. |
| Class | No | No | | A class is not generated yet, so it is skipped with a warning and a conforming type is written without it. |
| Unit | No | No | | A unit is not generated yet, so it is skipped with a warning and a quantity is written as its primitive. |
| Import and extern | No | No | | An extern is not generated yet, so a declaration naming one is skipped. |

<!-- /tdl:features -->
