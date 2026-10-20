# Zod

The `zod` backend is designed and not built yet: [zod-backend.md](../design/zod-backend.md) is its design, and [zod-backend-plan.md](../design/zod-backend-plan.md) orders its phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase of its plan that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features zod -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 1) | No | | |
| Mixin `include` | Planned (phase 1) | No | | |
| Fieldless enum | Planned (phase 1) | No | | |
| Fielded enum | Planned (phase 1) | No | | |
| Newtype | Planned (phase 1) | No | | |
| Alias | Planned (phase 1) | No | | |
| Optional and nullable | Planned (phase 1) | No | | |
| List, set, and map | Planned (phase 1) | No | | |
| `min` | Planned (phase 3) | No | | |
| `max` | Planned (phase 3) | No | | |
| `length` | Planned (phase 3) | No | | |
| `matches` | Planned (phase 3) | No | | |
| `oneOf` | Planned (phase 3) | No | | |
| `unique` | Planned (phase 3) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | Planned (phase 2) | No | | |
| `key` directive | No | No | | No phase plans it. |
| `name` directive | Planned (phase 1) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 1) | No | | |
| Type parameters | Planned (phase 5) | No | | |
| Class | Planned (phase 5) | No | | |
| Unit | Planned (phase 6) | No | | |
| Import and extern | Planned (phase 2) | No | | |

<!-- /tdl:features -->
