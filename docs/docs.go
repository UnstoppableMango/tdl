// Package docs embeds the specification, for a backend that hands it to a
// language model as the explanation of TDL.
package docs

import _ "embed"

// Spec is docs/spec.md.
//
//go:embed spec.md
var Spec string
