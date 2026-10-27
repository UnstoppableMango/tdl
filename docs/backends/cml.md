# Context Mapper

The `cml` backend is designed and not built yet: [#1267](https://github.com/UnstoppableMango/tdl/issues/1267) is its spec, and its sub-issues are the phases.
No phase is done.

## Features

What the backend will do with each feature of the language, and the phase that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features cml -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 1) | No | | |
| Mixin `include` | Planned (phase 1) | No | | |
| Fieldless enum | Planned (phase 1) | No | | |
| Fielded enum | Planned (phase 1) | No | | |
| Newtype | Planned (phase 1) | No | | |
| Alias | Planned (phase 1) | No | | |
| Optional and nullable | Planned (phase 1) | No | | |
| List, set, and map | Planned (phase 1) | No | | A map warns. |
| `min` | Planned (phase 1) | No | | |
| `max` | Planned (phase 1) | No | | |
| `length` | Planned (phase 1) | No | | |
| `matches` | Planned (phase 1) | No | | |
| `oneOf` | No | No | | CML has no option for it. |
| `unique` | No | No | | CML's `unique` is a unique column. |
| `owned` | Planned (phase 1) | No | | |
| Field default | No | No | | CML has no syntax for it. |
| `key` directive | Planned (phase 1) | No | | |
| `name` directive | Planned (phase 1) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 1) | No | | |
| Type parameters | No | No | | CML has no type parameters. |
| Class | No | No | | CML's `Trait` cannot hold one. |
| Unit | No | No | | |
| Import and extern | Planned (phase 3) | No | | |

<!-- /tdl:features -->
