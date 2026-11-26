package gen

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// Depend adds to a nondeterministic backend's request what it needs to
// refer to each import's dependency: the declaration each extern names,
// in Analysis.source's externs, and the files the dependency's own block
// for the target last wrote, in dependencies. The dependency's files are
// part of the request, so regenerating one changes the request's hash.
//
// It reads each dependency through loader, as lowering did, and its output
// through the dependency's `.tdl-output`. A dependency that cannot be read
// was reported by lowering; one with no block for the target, no out
// directive, or no output yet contributes nothing.
func Depend(req *plugin.Request, loader sema.Loader) error {
	model := req.GetModel()
	if len(model.GetImports()) == 0 || loader == nil {
		return nil
	}
	if req.Analysis == nil {
		req.Analysis = &plugin.Analysis{}
	}
	if req.Analysis.Source == nil {
		req.Analysis.Source = &plugin.Source{}
	}
	externs := make([]*plugin.Item, len(model.GetExterns()))
	for i := range externs {
		externs[i] = &plugin.Item{}
	}

	seen := map[string]bool{}
	for _, imp := range model.GetImports() {
		pkg := imp.GetPackage()
		if seen[pkg] {
			continue
		}
		name, src, err := loader.Load(imp.GetPosition().GetFilename(), imp.GetPath())
		if err != nil {
			continue
		}
		dep, err := parser.Parse(name, strings.NewReader(src))
		if err != nil {
			continue
		}
		seen[pkg] = true

		decls := map[string]ast.Decl{}
		for _, d := range dep.Decls {
			if analysis.IsDecl(d) && decls[d.Name()] == nil {
				decls[d.Name()] = d
			}
		}
		for i, ext := range model.GetExterns() {
			if d, ok := decls[ext.GetName()]; ok && ext.GetPackage() == pkg {
				externs[i] = &plugin.Item{Source: analysis.PrintItem(d), Name: ext.GetName()}
			}
		}

		files, err := dependencyFiles(req.GetOut(), req.GetTarget(), pkg, depOut(dep, name, req.GetTarget(), pkg))
		if err != nil {
			return fmt.Errorf("reading %s's output: %w", pkg, err)
		}
		req.Dependencies = append(req.Dependencies, files...)
	}
	req.Analysis.Source.Externs = externs
	return nil
}

// depOut is the out directive of dep's block for target, relative to the
// dependency's own file as the dependency's run reads it, or "".
func depOut(dep *ast.File, name, target, pkg string) string {
	for _, d := range dep.Decls {
		block, ok := d.(*ast.TargetDecl)
		if !ok || block.N != target || block.For != pkg {
			continue
		}
		for _, e := range block.Entries {
			if e.Path != "" || e.Directive == nil || e.Directive.N != host.OutDirective || len(e.Directive.Args) == 0 {
				continue
			}
			arg := e.Directive.Args[0]
			if arg.Kind != ast.LitString || arg.Text == "" {
				return ""
			}
			if filepath.IsAbs(arg.Text) {
				return arg.Text
			}
			return filepath.Join(filepath.Dir(name), arg.Text)
		}
	}
	return ""
}

// dependencyFiles reads the files target's record in depOut's marker
// lists, each with its path relative to out.
func dependencyFiles(out, target, pkg, depOut string) ([]*plugin.DependencyFile, error) {
	if depOut == "" {
		return nil, nil
	}
	kept, ok, err := Recorded(depOut, target)
	if err != nil || !ok {
		return nil, err
	}
	t, err := openTree(depOut)
	if err != nil {
		return nil, err
	}
	defer t.Close()

	var files []*plugin.DependencyFile
	for _, path := range kept.Paths() {
		rel, err := t.local(path, "read")
		if err != nil {
			return nil, err
		}
		content, err := t.readFile(rel)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		from, err := relative(out, path)
		if err != nil {
			return nil, err
		}
		files = append(files, &plugin.DependencyFile{Package: pkg, Path: filepath.ToSlash(from), Content: content})
	}
	return files, nil
}

// relative is path relative to dir, either of which may be relative to the
// working directory.
func relative(dir, path string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Rel(absDir, absPath)
}
