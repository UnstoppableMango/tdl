// Package gen is the compiler side of the plugin protocol: which backends
// exist, how a request is built, and what happens to the files that come
// back.
//
// It is private and free to change. The `plugin` package is the surface a
// backend author sees.
package gen

import (
	"sort"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/plugin"
)

// builtin maps a target name to a backend compiled into tdl.
//
// A name that is not here resolves to tdl-gen-<name> on PATH. Both kinds
// speak the same protocol; see docs/design/plugins.md.
var builtin = map[string]plugin.Backend{
	debug.Name: debug.Backend{},
}

// Builtin returns the backend compiled in under name.
func Builtin(name string) (plugin.Backend, bool) {
	b, ok := builtin[name]
	return b, ok
}

// BuiltinNames lists the compiled-in backends, sorted.
func BuiltinNames() []string {
	names := make([]string, 0, len(builtin))
	for name := range builtin {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
