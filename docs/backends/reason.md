# Reason

The `reason` backend is designed and not built yet: [#1297](https://github.com/UnstoppableMango/tdl/issues/1297) is its spec, and its sub-issues are the phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features reason -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 5) | No | | |
| Mixin `include` | Planned (phase 5) | No | | |
| Fieldless enum | Planned (phase 5) | No | | |
| Fielded enum | Planned (phase 5) | No | | |
| Newtype | Planned (phase 5) | No | | |
| Alias | Planned (phase 5) | No | | |
| Optional and nullable | Planned (phase 5) | No | | |
| List, set, and map | Planned (phase 5) | No | | |
| `min` | Planned (phase 5) | No | | |
| `max` | Planned (phase 5) | No | | |
| `length` | Planned (phase 5) | No | | |
| `matches` | No | No | | Warns and is not checked. |
| `oneOf` | Planned (phase 5) | No | | |
| `unique` | Planned (phase 5) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | No | No | | A field's default is not read. |
| `key` directive | Planned (phase 5) | No | | |
| `name` directive | Planned (phase 5) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 5) | No | | |
| Type parameters | Planned (phase 5) | No | | |
| Class | Planned (phase 5) | No | | |
| Unit | Planned (phase 5) | No | | A unit-typed field is its base type, with a warning. |
| Import and extern | Planned (phase 5) | No | | |

<!-- /tdl:features -->
