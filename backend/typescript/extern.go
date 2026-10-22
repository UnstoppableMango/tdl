package typescript

import (
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
)

// An extern is imported with `import type`, as backend/internal/ts
// describes: from the module a foreign directive names, or from the file
// its dependency's typescript block generates.

// externType is the TypeScript name of an extern, recording its import.
func (g *generator) externType(r *emit.Ref) (string, error) {
	return ts.Extern(g.Session, g.uses, r, ".js")
}
