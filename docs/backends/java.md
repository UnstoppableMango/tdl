# Java

The `java` backend is designed and not built yet: [java-backend.md](../design/java-backend.md) is its design, and [java-backend-plan.md](../design/java-backend-plan.md) orders its phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase of its plan that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features java -->

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
| `min` | Planned (phase 5) | No | | |
| `max` | Planned (phase 5) | No | | |
| `length` | Planned (phase 5) | No | | |
| `matches` | Planned (phase 5) | No | | |
| `oneOf` | Planned (phase 5) | No | | |
| `unique` | Planned (phase 5) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | Planned (phase 2) | No | | |
| `key` directive | Planned (phase 2) | No | | |
| `name` directive | Planned (phase 1) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 1) | No | | |
| Type parameters | Planned (phase 3) | No | | |
| Class | Planned (phase 4) | No | | |
| Unit | Planned (phase 7) | No | | |
| Import and extern | Planned (phase 2) | No | | |

<!-- /tdl:features -->
