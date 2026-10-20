# F\#

The `fsharp` backend is designed and not built yet: [ml-backends.md](../design/ml-backends.md) is its design, and [ml-backends-plan.md](../design/ml-backends-plan.md) orders its phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase of its plan that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features fsharp -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 7) | No | | |
| Mixin `include` | Planned (phase 7) | No | | |
| Fieldless enum | Planned (phase 7) | No | | |
| Fielded enum | Planned (phase 7) | No | | |
| Newtype | Planned (phase 7) | No | | |
| Alias | Planned (phase 7) | No | | |
| Optional and nullable | Planned (phase 7) | No | | |
| List, set, and map | Planned (phase 7) | No | | |
| `min` | Planned (phase 7) | No | | |
| `max` | Planned (phase 7) | No | | |
| `length` | Planned (phase 7) | No | | |
| `matches` | Planned (phase 7) | No | | |
| `oneOf` | Planned (phase 7) | No | | |
| `unique` | Planned (phase 7) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | No | No | | A field's default is not read. |
| `key` directive | Planned (phase 7) | No | | |
| `name` directive | Planned (phase 7) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 7) | No | | |
| Type parameters | Planned (phase 7) | No | | |
| Class | Planned (phase 7) | No | | |
| Unit | Planned (phase 7) | No | | |
| Import and extern | Planned (phase 7) | No | | |

<!-- /tdl:features -->
