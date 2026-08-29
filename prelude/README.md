# The standard prelude

No type is built in.
`std.tdl` declares the roots every other TDL file depends on, and a project may point `prelude` at a replacement directory.

The collection and optionality sugar resolves through these names: `[T]` is `List<T>`, `{T}` is `Set<T>`, `{K -> V}` is `Map<K, V>`, `T?` is `Option<T>`, and `T | null` is `Nullable<T>`.

`std.tdl` is embedded in the binary and loaded into an outer scope beneath every file, so a file may declare a name the prelude already has and its own wins.
Its declarations are merged into the model untagged: to a backend they are declarations like any other, which is what lets a replacement change what a collection is without any backend learning about it.

`tdl ir --prelude other.tdl` lowers against a different one, and `sema.WithPrelude` and `sema.WithoutPrelude` are the library equivalents.
Nothing in the compiler knows what `List` means; it knows only that `[T]` is spelled `List<T>`.
