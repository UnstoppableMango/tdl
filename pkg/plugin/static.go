package plugin

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
)

var Uml2Ts tdl.Plugin = NewAggregate(
	// fromPath{"uml2ts", 50},
	// github.NewUml2Ts(),
	docker{"ghcr.io/unstoppablemango/uml2ts", 75},
)

var static = []tdl.Plugin{Uml2Ts}
