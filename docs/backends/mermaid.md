# Mermaid

The `mermaid` backend is designed and not built yet: [#1482](https://github.com/UnstoppableMango/tdl/issues/1482) is its spec, and its sub-issues are the phases.
No phase is done.
It will be written in TypeScript on the JS/TS plugin SDK ([#1480](https://github.com/UnstoppableMango/tdl/issues/1480)) and served as `tdl-gen-mermaid`, a single executable that `tdl` runs from `PATH`.

It will draw a model as one Mermaid class diagram per package, in a `.mmd` file or, under `format("markdown")`, in a fenced block that GitHub renders.

## Features

What the backend will do with each feature of the language, and the phase that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features mermaid -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 1) | No | | Each is a class; a mixin is `<<mixin>>`. |
| Mixin `include` | Planned (phase 2) | No | | One inheritance edge per mixin. |
| Fieldless enum | Planned (phase 1) | No | | |
| Fielded enum | Planned (phase 1) | No | | A `<<union>>` class with one class per variant. |
| Newtype | Planned (phase 1) | No | | |
| Alias | Planned (phase 1) | No | | Expanded where used, with no class. |
| Optional and nullable | Planned (phase 1) | No | | |
| List, set, and map | Planned (phase 1) | No | | |
| `min` | Planned (phase 1) | No | | A diagram enforces nothing, so constraints are left out without a warning. |
| `max` | Planned (phase 1) | No | | |
| `length` | Planned (phase 1) | No | | |
| `matches` | Planned (phase 1) | No | | |
| `oneOf` | Planned (phase 1) | No | | |
| `unique` | Planned (phase 1) | No | | |
| `owned` | Planned (phase 2) | No | | A composition edge. |
| Field default | Planned (phase 1) | No | | Left out without a warning. |
| `key` directive | No | No | | A diagram has no identity keys. |
| `name` directive | No | No | | |
| `number` directive | No | No | | A diagram has no field numbers. |
| Doc comment | Planned (phase 1) | No | | Left out without a warning. |
| Type parameters | Planned (phase 1) | No | | |
| Class | Planned (phase 1) | No | | An `<<interface>>` class; phase 2 adds conformance edges. |
| Unit | Planned (phase 1) | No | | A unit has no class. |
| Import and extern | Planned (phase 2) | No | | A memberless class in its package's namespace. |

<!-- /tdl:features -->
