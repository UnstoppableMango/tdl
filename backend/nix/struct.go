package nix

import (
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// structure renders an entity, a value, or a mixin.
func (g *generator) structure(_ *strings.Builder, d *ir.Decl, _ string) error {
	return emit.Unsupported(d.GetMeta().GetPosition(), "%s is a struct, and structs are not generated yet", d.GetMeta().GetName())
}
