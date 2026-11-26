# Mermaid

The `mermaid` backend draws a model as one Mermaid class diagram per package, in a `.mmd` file or, under `format("markdown")`, in a fenced block of a `.md` file that GitHub renders.
It is a TypeScript plugin on the JS/TS plugin SDK, `tdl-gen-mermaid`, which `tdl gen` runs from `PATH`.
[mermaid-backend.md](../architecture/mermaid-backend.md) is its design, and [#1482](https://github.com/UnstoppableMango/tdl/issues/1482) tracks the phases: phases 1 and 2 draw classes, their members, and the edges between them, and the `direction`, `hide`, and `link` directives come in phase 3.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
A diagram enforces nothing, so constraints, defaults, and doc comments are left out without a warning.
No importer is planned, so every Import cell is No.

<!-- tdl:features mermaid -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a class; a mixin is `<<mixin>>` | |
| Mixin `include` | Yes | No | an inheritance edge from the mixin, whose fields it no longer lists | One edge per mixin, however many fields it adds. |
| Fieldless enum | Yes | No | an `<<enumeration>>` class, one member per variant | |
| Fielded enum | Yes | No | a `<<union>>` class with a class per variant, each an inheritance edge from it | Variant classes are named `Enum_Variant` and labelled with the variant's name. |
| Newtype | Yes | No | a `<<newtype>>` class whose one member is its base | |
| Alias | Yes | No | expanded where used, with no class | |
| Optional and nullable | Yes | No | `Option~T~`; a nullable union is written as text | |
| List, set, and map | Yes | No | `List~T~`, `Set~T~`, and `Map~K,V~` | A field draws an edge to each declaration it names, its far end `*`. |
| `min` | Yes | No | | A diagram enforces nothing, so constraints are left out without a warning. |
| `max` | Yes | No | | Left out without a warning. |
| `length` | Yes | No | | Left out without a warning. |
| `matches` | Yes | No | | Left out without a warning. |
| `oneOf` | Yes | No | | Left out without a warning. |
| `unique` | Yes | No | | Left out without a warning. |
| `owned` | Yes | No | a composition edge, `*--` | An unowned field is an association, `-->`. |
| Field default | Yes | No | | Left out without a warning. |
| `key` directive | No | No | | The backend declares no `key` directive, since a diagram has no identity keys. |
| `name` directive | No | No | | The backend declares no `name` directive. |
| `number` directive | No | No | | The backend declares no `number` directive, since a diagram has no field numbers. |
| Doc comment | Yes | No | | Left out without a warning. |
| Type parameters | Yes | No | `class Page~T~`, and `Page~T~` where applied | |
| Class | Yes | No | an `<<interface>>` class | A conforming declaration draws a dotted realization edge to each of the model's classes. |
| Unit | Yes | No | the unit as an argument, `decimal~kg~` | A unit has no class. |
| Import and extern | Yes | No | the extern's name | A field naming an extern draws it as a memberless class in its package's namespace, `namespace acme_money { }`. |

<!-- /tdl:features -->
