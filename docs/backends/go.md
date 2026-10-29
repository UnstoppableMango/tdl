# Go

The `go` backend writes structs, entity keys, both enum shapes, newtypes, generics, classes as interfaces, `Validate` methods, and foreign types, one file per declaration or one for the whole target.
[go-backend.md](../architecture/go-backend.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back.

<!-- tdl:features go -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | Warns (`lossy.struct-kind`) | a `struct` | An entity reads back as one only when a `key` directive gives it a `Key` method, and a mixin reads back as a value. |
| Mixin `include` | Yes | Warns (`lossy.include`) | the mixin's fields, copied in | Reads back as fields of the struct, without the include. |
| Fieldless enum | Yes | Yes | a `string` type with constants | |
| Fielded enum | Yes | Yes | a sealed interface with a struct per variant | |
| Newtype | Yes | Yes | a defined type, such as `type CartId string` | |
| Alias | Yes | Warns (`lossy.alias`) | expanded where used | Reads back as the type it expands to. |
| Optional and nullable | Yes | Warns (`lossy.optional`) | a pointer, `*T` | An optional and a nullable value are both a pointer, so a nullable value reads back as `T?`. |
| List, set, and map | Yes | Yes | `[]T`, `map[K]struct{}`, and `map[K]V` | |
| `min` | Yes | Yes | a check in `Validate` | |
| `max` | Yes | Yes | a check in `Validate` | |
| `length` | Yes | Yes | a check in `Validate`, counting runes | |
| `matches` | Yes | Yes | a `regexp` check in `Validate` | |
| `oneOf` | Yes | Yes | a check in `Validate` | |
| `unique` | Yes | Yes | a check in `Validate` for repeated elements | |
| `owned` | Warns (`lossy.owned`) | Warns (`lossy.owned`) | | Go does not carry ownership, and the field holds what it names by value. |
| Field default | Warns (`lossy.default`) | Warns (`lossy.default`) | | Go has no field defaults, so the zero value stands. |
| `key` directive | Yes | Yes | a `Key()` method | A key of several fields returns a `<Name>Key` struct. |
| `name` directive | Yes | Warns (`lossy.name`) | the identifier, as written | The name is used verbatim, so `grandTotal` is an unexported field, and it reads back as the field's own name. |
| `number` directive | No | No | | Go has no field numbers, and the backend does not declare the directive, so it warns without a loss code and is ignored. |
| Doc comment | Yes | Yes | a `//` doc comment | |
| Type parameters | Yes | Yes | type parameters, constrained by `any` or `comparable` | |
| Class | Warns (`lossy.class`) | Warns (`lossy.class`) | an interface with an unexported marker method | The class's fields are not written, since generated code never reads them. |
| Unit | Planned (phase 6) | No | | Go has no units, so the struct holding a unit-typed field is skipped with a warning, and import has nothing to read back. |
| Import and extern | Yes | No | an import of the dependency's package | Generating needs the dependency's `go` block to name its `package`; import reads the extern back as a local placeholder with a `foreign` directive, and no loss code says so. |

<!-- /tdl:features -->
