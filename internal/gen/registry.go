// Package gen is the compiler side of the plugin protocol: which backends
// exist, how a request is built, and what happens to the files that come
// back.
//
// It is private and free to change. The `plugin` package is the surface a
// backend author sees.
package gen

import (
	"maps"
	"slices"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/backend/graphql"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/backend/thrift"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/plugin"
)

// builtin maps a target name to a backend compiled into tdl.
//
// A name that is not here resolves to tdl-gen-<name> on PATH. Both kinds
// speak the same protocol; see docs/design/plugins.md.
var builtin = map[string]plugin.Backend{
	debug.Name:      debug.Backend{},
	golang.Name:     golang.Backend{},
	graphql.Name:    graphql.Backend{},
	protobuf.Name:   protobuf.Backend{},
	smithy.Name:     smithy.Backend{},
	thrift.Name:     thrift.Backend{},
	typescript.Name: typescript.Backend{},
}

// Builtin returns the backend compiled in under name.
func Builtin(name string) (plugin.Backend, bool) {
	b, ok := builtin[name]
	return b, ok
}

// Resolve returns the backend serving a target: the compiled-in one if
// there is one, otherwise tdl-gen-<name> on PATH.
//
// Both speak the same protocol, so everything above this treats them the
// same. Preferring the built-in means a name tdl ships cannot be shadowed
// by whatever happens to be on PATH.
func Resolve(name string) (plugin.Backend, error) {
	if b, ok := Builtin(name); ok {
		return b, nil
	}
	return Find(name)
}

// BuiltinNames lists the compiled-in backends, sorted.
func BuiltinNames() []string {
	return slices.Sorted(maps.Keys(builtin))
}
