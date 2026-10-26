# Zod

The `zod` backend writes one TypeScript module of [Zod](https://zod.dev) schemas per model, each parsing a declaration's JSON wire value and inferring the type the `typescript` backend writes for it.
The output imports `zod`, 4.3 or later, which is the consumer's dependency.
[zod-backend.md](../architecture/zod-backend.md) is its design; phases 1 and 2 are built, and [#1230](https://github.com/UnstoppableMango/tdl/issues/1230) tracks the rest.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`, and the phase that builds what it does not do yet.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
No importer is planned, so every Import cell is No.

<!-- tdl:features zod -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | `z.object` | Entities, values, and mixins are all objects; a `closed` directive makes one a `z.strictObject`. |
| Mixin `include` | Yes | No | the mixin's properties, copied in | |
| Fieldless enum | Yes | No | `z.enum` of the variants' names | |
| Fielded enum | Yes | No | a `z.discriminatedUnion` on `kind` | A `discriminant` directive renames `kind`. |
| Newtype | Yes | No | a schema of its base | Not branded until phase 3. |
| Alias | Yes | No | expanded where used | |
| Optional and nullable | Yes | No | `.exactOptional()` or `.nullable()` | |
| List, set, and map | Yes | No | `z.array`, a uniqueness `refine`, and `z.record` | A map keyed by a fieldless enum is a `z.partialRecord`. |
| `min` | Planned (phase 3) | No | | |
| `max` | Planned (phase 3) | No | | |
| `length` | Planned (phase 3) | No | | |
| `matches` | Planned (phase 3) | No | | Checked only under a `patterns` directive. |
| `oneOf` | Planned (phase 3) | No | | |
| `unique` | Planned (phase 3) | No | | |
| `owned` | Yes | No | | Not written: a schema has no ownership, and a referenced entity is parsed by value. |
| Field default | Yes | No | `.default(value)` | Checked against the schema when generated, since Zod does not check it; one the schema refuses warns and is not written. |
| `key` directive | No | No | | The backend declares no `key` directive, since a schema has no identity. |
| `name` directive | Yes | No | the property or schema name | |
| `number` directive | No | No | | JSON has no field numbers. |
| Doc comment | Yes | No | a JSDoc comment on the schema, its type, and each property | |
| Type parameters | Planned (phase 5) | No | | |
| Class | Planned (phase 5) | No | | |
| Unit | Planned (phase 6) | No | | Until then a quantity is its base type, with a warning. |
| Import and extern | Yes | No | an `import` from the dependency's module | Placed by both blocks' `out` directives, or named by a `foreign` directive. |

<!-- /tdl:features -->
