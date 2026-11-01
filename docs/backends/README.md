# Backends

Each page says what a backend generates and what it does with each feature of the language: generated, generated with a warning that names the loss, or skipped.
The cells are measured by `make features`, so a page cannot claim more than the backend does.

| Backend | Target |
| --- | --- |
| [`go`](go.md) | Go |
| [`protobuf`](protobuf.md) | Protocol Buffers |
| [`thrift`](thrift.md) | Thrift |
| [`smithy`](smithy.md) | Smithy |
| [`graphql`](graphql.md) | GraphQL |
| [`typescript`](typescript.md) | TypeScript |
| [`jsonschema`](jsonschema.md) | JSON Schema |
| [`openapi`](openapi.md) | OpenAPI |
| [`salesforce`](salesforce.md) | Salesforce |
| [`haskell`](haskell.md) | Haskell |
| [`csharp`](csharp.md) | C# |
| [`likec4`](likec4.md) | LikeC4 |
| [`zod`](zod.md) | Zod |

These are designed and not built yet, and each page names the phase that builds each feature:

| Backend | Target |
| --- | --- |
| [`java`](java.md) | Java |
| [`python`](python.md) | Python |
| [`rust`](rust.md) | Rust |
| [`ocaml`](ocaml.md) | OCaml |
| [`reason`](reason.md) | Reason |
| [`sml`](sml.md) | Standard ML |
| [`fsharp`](fsharp.md) | F# |
| [`cml`](cml.md) | Context Mapper |

## Import

What `tdl import` reads back of each feature, for the backends that import: each generates a model using the feature without `roundtrip` annotations, and imports the files it wrote.
Yes means reading back adds no loss: the same model, or one differing only by the output loss Generate reports. Warns reads back less under a loss code, and No reads back a different model without one, or fails.
A backend that imports nothing has no column.

<!-- tdl:features import -->

| Feature | `go` | `graphql` | `protobuf` | `smithy` | `thrift` | `typescript` |
| --- | --- | --- | --- | --- | --- | --- |
| Entity, value, and mixin | Warns | Warns | Warns | Warns | Warns | Warns |
| Mixin `include` | Warns | Warns | Warns | Warns | Warns | Warns |
| Fieldless enum | Yes | Yes | Yes | Yes | Yes | Yes |
| Fielded enum | Yes | Yes | Yes | Yes | Yes | Yes |
| Newtype | Yes | Yes | Yes | Yes | Yes | Yes |
| Alias | Warns | Warns | Warns | Warns | Warns | Warns |
| Optional and nullable | Warns | Warns | Warns | Warns | Warns | Yes |
| List, set, and map | Yes | No | Yes | Yes | Yes | Yes |
| `min` | Yes | Yes | Yes | Yes | Yes | Yes |
| `max` | Yes | Yes | Yes | Yes | Yes | Yes |
| `length` | Yes | Yes | Yes | Yes | Yes | Yes |
| `matches` | Yes | Yes | Yes | Yes | Yes | Yes |
| `oneOf` | Yes | Yes | Yes | Yes | Yes | Yes |
| `unique` | Yes | Yes | Yes | Yes | Yes | Yes |
| `owned` | Yes | Yes | Yes | Yes | Yes | Yes |
| Field default | Yes | Yes | Yes | Yes | Yes | Yes |
| `key` directive | Yes | No | No | No | No | No |
| `name` directive | Warns | Warns | Warns | Warns | Warns | Warns |
| `number` directive | No | No | Yes | No | Yes | No |
| Doc comment | Yes | Yes | Yes | Yes | Yes | Yes |
| Type parameters | Yes | No | No | No | No | No |
| Class | Yes | Yes | Yes | Yes | Yes | Yes |
| Unit | No | Yes | Yes | Yes | Yes | Yes |
| Import and extern | No | No | No | No | No | Yes |

<!-- /tdl:features -->
