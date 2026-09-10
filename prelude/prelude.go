// Package prelude embeds the standard prelude, the TDL source declaring the
// roots every other file depends on.
//
// No type is built in. `[T]` is `List<T>` and `T?` is `Option<T>` only
// because something declares `List` and `Option`, and a project may point
// at a replacement that declares them differently.
package prelude

import _ "embed"

// Name is what the standard prelude is called in diagnostics.
const Name = "std.tdl"

// Source is the standard prelude.
//
//go:embed std.tdl
var Source string
