package plugin

import tdl "github.com/unstoppablemango/tdl/pkg"

var Uml2Ts tdl.Plugin = NewAggregate(
	fromPath{"uml2ts", 50},
)
