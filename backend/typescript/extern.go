package typescript

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
)

// An extern is imported with `import type`, as backend/internal/ts
// describes: from the module a foreign directive names, or from the file
// its dependency's typescript block generates. A mapped module imports one
// without a foreign directive from its .tdl file instead.

// externType is the TypeScript name of an extern, recording its import.
func (g *generator) externType(r *emit.Ref) (string, error) {
	if _, foreign := g.Find(r.Extern.GetDirectives(), "foreign"); g.mapped && !foreign {
		module, err := g.sourceModule(r)
		if err != nil {
			return "", err
		}
		return g.uses.Add(r.Name, module, ts.ExternName(g.Session, r), r.Pos)
	}
	return ts.Extern(g.Session, g.uses, r, ".js")
}

// sourceModule is the specifier a mapped module imports an extern by: the
// .tdl file declaring it, by the path this file imports it with, which
// tsc sends back through the mapper.
func (g *generator) sourceModule(r *emit.Ref) (string, error) {
	var paths []string
	for _, dep := range g.Model.GetImports() {
		if dep.GetPackage() == r.Extern.GetPackage() && !slices.Contains(paths, dep.GetPath()) {
			paths = append(paths, dep.GetPath())
		}
	}
	switch len(paths) {
	case 0:
		return "", emit.Unsupported(r.Pos, "%s is declared in package %s, which this file does not import", r.Name, r.Extern.GetPackage())
	case 1:
	default:
		return "", emit.Unsupported(r.Pos, "%s is declared in package %s, which %s all declare, so the module it comes from is unknown",
			r.Name, r.Extern.GetPackage(), strings.Join(paths, ", "))
	}
	spec := path.Clean(filepath.ToSlash(paths[0]))
	if !strings.HasPrefix(spec, "../") && !strings.HasPrefix(spec, "/") {
		spec = "./" + spec
	}
	return spec, nil
}
