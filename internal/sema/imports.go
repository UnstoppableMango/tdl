package sema

import (
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// loadImports walks the import graph from file, recording each import in
// the model and reporting cycles.
//
// A dependency is parsed but not lowered. What the walk needs from it is
// its package name, its own imports, and, for a `_` import, the names it
// exports. Whether a qualified reference names something that dependency
// actually declares is not checked here: the reference carries the
// dependency's package to the backend, which is what ir.md asks for.
func (l *lowerer) loadImports(file *ast.File) {
	if len(file.Imports) == 0 {
		return
	}
	if l.loader == nil {
		for _, imp := range file.Imports {
			l.diags.add(imp.P, "imports need a loader: %q", imp.Path)
		}
		return
	}

	l.walkImports(file, file.Filename, map[string]bool{file.Filename: true}, []string{file.Filename}, true)
}

// walkImports records file's imports and recurses. `root` says whether
// these imports are the ones the model should list: a dependency's own
// imports are walked for cycle detection but do not appear in this model.
func (l *lowerer) walkImports(file *ast.File, from string, onPath map[string]bool, chain []string, root bool) {
	for _, imp := range file.Imports {
		name, src, err := l.loader.Load(from, imp.Path)
		if err != nil {
			l.diags.add(imp.P, "cannot read import %q: %v", imp.Path, err)
			continue
		}

		if onPath[name] {
			l.diags.add(imp.P, "import cycle: %s", strings.Join(append(chain, name), " -> "))
			continue
		}

		dep, perr := parser.Parse(name, strings.NewReader(src))
		if perr != nil {
			l.diags.add(imp.P, "import %q does not parse: %v", imp.Path, perr)
			continue
		}

		pkg := ""
		if dep.Package != nil {
			pkg = dep.Package.Path
		}

		if root {
			l.model.Imports = append(l.model.Imports, &ir.Import{
				Path:     imp.Path,
				Alias:    imp.Alias,
				Package:  pkg,
				Position: position(imp.P),
			})
			l.bindImport(imp, pkg, dep)
		}

		onPath[name] = true
		l.walkImports(dep, name, onPath, append(chain, name), false)
		delete(onPath, name)
	}
}

// bindImport binds what an import brings into scope.
//
// An aliased import binds the alias to a package, and a qualified
// reference through it becomes an extern. A `_` import merges the
// dependency's exported names directly, so it has to know what they are:
// nothing else can tell what a bare name in this file refers to.
func (l *lowerer) bindImport(imp *ast.ImportDecl, pkg string, dep *ast.File) {
	if imp.Alias != "_" {
		if prev, ok := l.aliases[imp.Alias]; ok {
			l.diags.add(imp.P, "import alias %s is bound twice, first to %s", imp.Alias, prev)
			return
		}
		l.aliases[imp.Alias] = pkg
		return
	}

	for _, decl := range dep.Decls {
		name := decl.Name()
		if !exported(name) || !namesAType(decl) {
			continue
		}
		if _, ok := l.file.bind(name, binding{
			kind: bindExtern,
			id:   l.extern(pkg, name, imp.P),
			pos:  imp.P,
		}); !ok {
			l.diags.add(imp.P, "%s from %q is already declared here", name, imp.Path)
		}
	}
}

// exported reports whether a declaration is visible outside its package. A
// name beginning with an upper-case letter is exported, and everything else
// is package-private.
func exported(name string) bool {
	if name == "" {
		return false
	}
	r := name[0]
	return r >= 'A' && r <= 'Z'
}

// extern returns the ID of a foreign declaration, adding it to the table
// only if it is not already there.
func (l *lowerer) extern(pkg, name string, pos ast.Position) *ir.ID {
	key := pkg + "." + name
	if idx, ok := l.externs[key]; ok {
		return &ir.ID{Index: idx, Name: key}
	}

	idx := int32(len(l.model.Externs))
	l.externs[key] = idx
	l.model.Externs = append(l.model.Externs, &ir.Extern{
		Package:  pkg,
		Name:     name,
		Position: position(pos),
	})
	return &ir.ID{Index: idx, Name: key}
}
