package emit

import (
	"fmt"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Loss codes name a fact a conversion cannot carry, in either direction.
// A warning carrying one can be silenced in tdl.toml or with --allow-lossy.
// docs/design/reverse.md describes each.
const (
	LossPrimitive   = "lossy.primitive"
	LossNewtype     = "lossy.newtype"
	LossCollection  = "lossy.collection"
	LossOptional    = "lossy.optional"
	LossStructKind  = "lossy.struct-kind"
	LossInclude     = "lossy.include"
	LossAlias       = "lossy.alias"
	LossConstraint  = "lossy.constraint"
	LossOwned       = "lossy.owned"
	LossDefault     = "lossy.default"
	LossKey         = "lossy.key"
	LossName        = "lossy.name"
	LossNumber      = "lossy.number"
	LossOrder       = "lossy.order"
	LossDoc         = "lossy.doc"
	LossGeneric     = "lossy.generic"
	LossClass       = "lossy.class"
	LossUnit        = "lossy.unit"
	LossUnsupported = "lossy.unsupported"
)

// LossCodes lists every loss code, in the order the design lists them.
var LossCodes = []string{
	LossPrimitive, LossNewtype, LossCollection, LossOptional, LossStructKind,
	LossInclude, LossAlias, LossConstraint, LossOwned, LossDefault, LossKey,
	LossName, LossNumber, LossOrder, LossDoc, LossGeneric, LossClass,
	LossUnit, LossUnsupported,
}

// Lossy warns that a fact is lost, with the code that names it.
func (s *Session) Lossy(code string, pos *ir.Position, format string, args ...any) {
	s.Diags = append(s.Diags, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
		Code:     code,
	})
}
