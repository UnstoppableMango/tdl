# JSON Schema

The `jsonschema` backend writes one JSON Schema document per model: definitions, both enum shapes, and constraints as keywords.
[schema-backends.md](../design/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

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
