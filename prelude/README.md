# The standard prelude

No type is built in.
`std.tdl` declares the primitives, collections, options, SI base units, and the `Entity` class that every file depends on.

The sugar resolves through its names: `[T]` is `List<T>`, `{T}` is `Set<T>`, `{K -> V}` is `Map<K, V>`, `T?` is `Option<T>`, and `T | null` is `Nullable<T>`.
The compiler knows those spellings and nothing about what the types mean.

`std.tdl` is embedded in the binary and loaded in a scope beneath every file, so a file's declaration of the same name wins.
Its declarations reach backends like any other, so a replacement prelude can change what a collection is without a backend change.
Use one with `tdl ir --prelude other.tdl`, or `sema.WithPrelude` from Go.
