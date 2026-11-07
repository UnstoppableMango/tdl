package emit

import (
	"fmt"

	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Loss codes name a fact a conversion cannot carry, in either direction.
// internal/host declares them, so a host outside backend/ reads the same
// list; docs/architecture/reverse.md describes each.
const (
	LossPrimitive   = host.LossPrimitive
	LossNewtype     = host.LossNewtype
	LossCollection  = host.LossCollection
	LossOptional    = host.LossOptional
	LossStructKind  = host.LossStructKind
	LossInclude     = host.LossInclude
	LossAlias       = host.LossAlias
	LossConstraint  = host.LossConstraint
	LossOwned       = host.LossOwned
	LossDefault     = host.LossDefault
	LossKey         = host.LossKey
	LossName        = host.LossName
	LossNumber      = host.LossNumber
	LossOrder       = host.LossOrder
	LossDoc         = host.LossDoc
	LossGeneric     = host.LossGeneric
	LossClass       = host.LossClass
	LossUnit        = host.LossUnit
	LossUnsupported = host.LossUnsupported
)

// LossCodes lists every loss code, in the order the design lists them.
var LossCodes = host.LossCodes

// Lossy warns that a fact is lost, with the code that names it. It reports
// output loss: what the output itself drops or weakens, which annotations
// do not restore, so it warns under roundtrip too. [Session.Lose] reports
// read-back loss.
func (s *Session) Lossy(code string, pos *ir.Position, format string, args ...any) {
	s.Diags = append(s.Diags, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
		Code:     code,
	})
}
