# OpenAPI

The `openapi` backend writes one OpenAPI 3.1, 3.0, or 2.0 document per model, of schemas in YAML or JSON: both enum shapes (no fielded enums in 2.0) and constraints as keywords, but no paths.
[schema-backends.md](../architecture/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
