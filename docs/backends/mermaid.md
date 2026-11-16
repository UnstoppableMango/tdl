# Mermaid

The `mermaid` backend draws a model as one Mermaid class diagram per package, in a `.mmd` file or, under `format("markdown")`, in a fenced block of a `.md` file that GitHub renders.
It is a TypeScript plugin on the JS/TS plugin SDK, `tdl-gen-mermaid`, which `tdl gen` runs from `PATH`.
It ships as a single executable beside `tdl` in the Nix package, the release tarballs, and the npm platform packages, so it needs no JS runtime.
[mermaid-backend.md](../architecture/mermaid-backend.md) is its design, and [#1482](https://github.com/UnstoppableMango/tdl/issues/1482) tracks the phases: phase 1 draws classes and their members, and edges between declarations come in phase 2.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
A diagram enforces nothing, so constraints, defaults, and doc comments are left out without a warning.
No importer is planned, so every Import cell is No.

<!-- tdl:features mermaid -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a class; a mixin is `<<mixin>>` | |
| Mixin `include` | Yes | No | the mixin's fields as members | Phase 2 draws one inheritance edge per mixin. |
| Fieldless enum | Yes | No | an `<<enumeration>>` class, one member per variant | |
| Fielded enum | Yes | No | a `<<union>>` class with a class per variant, each an inheritance edge from it | Variant classes are named `Enum_Variant` and labelled with the variant's name. |
| Newtype | Yes | No | a `<<newtype>>` class whose one member is its base | |
| Alias | Yes | No | expanded where used, with no class | |
| Optional and nullable | Yes | No | `Option~T~`; a nullable union is written as text | |
| List, set, and map | Yes | No | `List~T~`, `Set~T~`, and `Map~K,V~` | Phase 2 adds an edge to each declaration a field names. |
| `min` | Yes | No | | A diagram enforces nothing, so constraints are left out without a warning. |
| `max` | Yes | No | | Left out without a warning. |
| `length` | Yes | No | | Left out without a warning. |
| `matches` | Yes | No | | Left out without a warning. |
| `oneOf` | Yes | No | | Left out without a warning. |
| `unique` | Yes | No | | Left out without a warning. |
| `owned` | Yes | No | | Phase 2 draws a composition edge. |
| Field default | Yes | No | | Left out without a warning. |
| `key` directive | No | No | | The backend declares no `key` directive, since a diagram has no identity keys. |
| `name` directive | No | No | | The backend declares no `name` directive. |
| `number` directive | No | No | | The backend declares no `number` directive, since a diagram has no field numbers. |
| Doc comment | Yes | No | | Left out without a warning. |
| Type parameters | Yes | No | `class Page~T~`, and `Page~T~` where applied | |
| Class | Yes | No | an `<<interface>>` class | Phase 2 draws a realization edge to each conformance. |
| Unit | Yes | No | the unit as an argument, `decimal~kg~` | A unit has no class. |
| Import and extern | Yes | No | the extern's name | Phase 2 draws the extern as a class in its package's namespace. |

<!-- /tdl:features -->
