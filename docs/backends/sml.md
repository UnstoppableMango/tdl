# Standard ML

The `sml` backend is designed and not built yet: [#1297](https://github.com/UnstoppableMango/tdl/issues/1297) is its spec, and its sub-issues are the phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features sml -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 6) | No | | |
| Mixin `include` | Planned (phase 6) | No | | |
| Fieldless enum | Planned (phase 6) | No | | |
| Fielded enum | Planned (phase 6) | No | | |
| Newtype | Planned (phase 6) | No | | |
| Alias | Planned (phase 6) | No | | |
| Optional and nullable | Planned (phase 6) | No | | |
| List, set, and map | Planned (phase 6) | No | | |
| `min` | Planned (phase 6) | No | | |
| `max` | Planned (phase 6) | No | | |
| `length` | Planned (phase 6) | No | | |
| `matches` | No | No | | Warns and is not checked. |
| `oneOf` | Planned (phase 6) | No | | |
| `unique` | Planned (phase 6) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | No | No | | A field's default is not read. |
| `key` directive | Planned (phase 6) | No | | |
| `name` directive | Planned (phase 6) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 6) | No | | |
| Type parameters | Planned (phase 6) | No | | |
| Class | Planned (phase 6) | No | | |
| Unit | Planned (phase 6) | No | | A unit-typed field is its base type, with a warning. |
| Import and extern | Planned (phase 6) | No | | |

<!-- /tdl:features -->
