package sema

import (
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// loadImports walks the import graph from file, recording each import in
// the model and reporting cycles. A dependency is parsed but not lowered.
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

// walkImports records file's imports and recurses. Only `root` imports are
// listed in the model; a dependency's are walked for cycle detection.
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
				Path:       imp.Path,
				Alias:      imp.Alias,
				Package:    pkg,
				Position:   position(imp.P),
				Directives: l.depDirectives(dep, pkg),
			})
			l.bindImport(imp, pkg, dep)
		}

		onPath[name] = true
		l.walkImports(dep, name, onPath, append(chain, name), false)
		delete(onPath, name)
	}
}

// depDirectives collects the block-scope directives of a dependency's
// target blocks for its package: the bare directives at a block's top
// level.
func (l *lowerer) depDirectives(dep *ast.File, pkg string) []*ir.Directive {
	var out []*ir.Directive
	for _, decl := range dep.Decls {
		block, ok := decl.(*ast.TargetDecl)
		if !ok || block.For != pkg {
			continue
		}
		for _, entry := range block.Entries {
			if entry.Entries == nil && entry.Path == "" {
				out = append(out, l.directive(block.N, entry.Directive))
			}
		}
	}
	return out
}

// bindImport binds what an import brings into scope: an alias to a
// package, or for a `_` import, the dependency's exported names.
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
		if !exported(decl) || !namesAType(decl) {
			continue
		}
		// The position is the declaration's, in the dependency.
		if _, ok := l.file.bind(name, binding{
			kind: bindExtern,
			id:   l.extern(pkg, name, imp.P),
			pos:  decl.Pos(),
		}); !ok {
			l.diags.add(imp.P, "%s from %q is already declared here", name, imp.Path)
		}
	}
}

// exported reports whether decl is visible outside its package: an
// upper-case name, or any primitive or unit declaration.
func exported(decl ast.Decl) bool {
	switch decl.(type) {
	case *ast.PrimitiveDecl, *ast.UnitDecl:
		return true
	}
	name := decl.Name()
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
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
