// Package prelude embeds the standard prelude, the TDL source declaring
// every type, including `List` and `Option`, that sugar such as `[T]` and
// `T?` names.
package prelude

import _ "embed"

// Name is what the standard prelude is called in diagnostics.
const Name = "std.tdl"

// Source is the standard prelude.
//
//go:embed std.tdl
var Source string
