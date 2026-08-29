# The standard prelude

No type is built in.
`std.tdl` declares the roots every other TDL file depends on, and a project may point `prelude` at a replacement directory.

The collection and optionality sugar resolves through these names: `[T]` is `List<T>`, `{T}` is `Set<T>`, `{K -> V}` is `Map<K, V>`, `T?` is `Option<T>`, and `T | null` is `Nullable<T>`.

The `Entity` and `Value` classes are not here yet.
They arrive with phase 4 of [../docs/design/parser-plan.md](../docs/design/parser-plan.md).

Nothing loads this file yet.
Making it a real compilation unit is phase 4 of [../docs/design/ir-plan.md](../docs/design/ir-plan.md); until then it is the target the parser is built against.
