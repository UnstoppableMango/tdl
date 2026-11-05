# Nix

The `nix` backend is designed and not built yet: [#1437](https://github.com/UnstoppableMango/tdl/issues/1437) is its spec, and its sub-issues are the phases.
No phase is done.

It will write a model as nixpkgs `lib.types` option types, one file per model, whose values `builtins.toJSON` writes in the JSON wire convention.

## Features

What the backend will do with each feature of the language, and the phase that builds it.
No means no phase plans it.
No importer is planned, so every Import cell is No.
Once the backend is built, `make features` measures this table like every other.

<!-- tdl:features nix -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Planned (phase 1) | No | | Every kind is a submodule. |
| Mixin `include` | Planned (phase 1) | No | | |
| Fieldless enum | Planned (phase 1) | No | | |
| Fielded enum | Planned (phase 2) | No | | |
| Newtype | Planned (phase 2) | No | | |
| Alias | Planned (phase 2) | No | | |
| Optional and nullable | Planned (phase 1) | No | | |
| List, set, and map | Planned (phase 1) | No | | |
| `min` | Planned (phase 3) | No | | |
| `max` | Planned (phase 3) | No | | |
| `length` | Planned (phase 3) | No | | |
| `matches` | Planned (phase 3) | No | | A pattern POSIX ERE cannot state will warn. |
| `oneOf` | Planned (phase 3) | No | | |
| `unique` | Planned (phase 3) | No | | |
| `owned` | No | No | | No phase plans it. |
| Field default | Planned (phase 2) | No | | |
| `key` directive | No | No | | The target has no methods. |
| `name` directive | Planned (phase 1) | No | | |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Planned (phase 1) | No | | |
| Type parameters | Planned (phase 4) | No | | |
| Class | Planned (phase 1) | No | | Will warn: conformance checks nothing at evaluation. |
| Unit | Planned (phase 1) | No | | Will warn: a unit-typed field is a number. |
| Import and extern | Planned (phase 4) | No | | |

<!-- /tdl:features -->
