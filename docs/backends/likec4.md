# LikeC4

The `likec4` backend writes LikeC4 source for an architecture diagram: an element for each structure, enum, newtype, and class.
[likec4-backend.md](../design/likec4-backend.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

<!-- tdl:features likec4 -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | an `entity`, `value`, or `mixin` element | |
| Mixin `include` | Yes | No | an `- include` line in the description | Phase 2 adds an `includes` relationship. |
| Fieldless enum | Yes | No | an `enum` element, its variants in the description | |
| Fielded enum | Yes | No | an `enum` element, its variants in the description | |
| Newtype | Yes | No | a `newtype` element, its base as `technology` | |
| Alias | Yes | No | expanded where used, with no element | |
| Optional and nullable | Yes | No | the TDL type in the description | |
| List, set, and map | Yes | No | the TDL type in the description | Phase 2 adds a relationship to each declaration a field names. |
| `min` | Yes | No | listed in the field's description line | A diagram enforces nothing, so a constraint is listed rather than warned about. |
| `max` | Yes | No | listed in the field's description line | |
| `length` | Yes | No | listed in the field's description line | |
| `matches` | Yes | No | listed in the field's description line | |
| `oneOf` | Yes | No | listed in the field's description line | |
| `unique` | Yes | No | listed in the field's description line | |
| `owned` | Yes | No | `owned` in the field's description line | Phase 2 adds a `composes` relationship. |
| Field default | Yes | No | `= value` in the field's description line | |
| `key` directive | No | No | | The backend declares no `key` directive, since a diagram has no identity keys. |
| `name` directive | No | No | | Phase 4 adds `name` on a declaration, as its element id; a field has no id to rename. |
| `number` directive | No | No | | The backend declares no `number` directive, since a diagram has no field numbers. |
| Doc comment | Yes | No | the element's `description` | A field's or a variant's doc comment is not drawn. |
| Type parameters | Yes | No | the signature, such as `Page<T>`, as `technology` | |
| Class | Yes | No | a `class` element | Phase 3 adds a `conforms` relationship. |
| Unit | Yes | No | the TDL type in the description | A unit has no element. |
| Import and extern | Yes | No | the extern's name in the description | Phase 4 adds a relationship into the dependency. |

<!-- /tdl:features -->
