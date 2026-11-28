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
// through the dependency's `.tdl-output`, returning every marker and file
// it read, for `--make`. A dependency that cannot be read was reported by
// lowering; one with no block for the target, no out directive, an out
// that is the request's own, or no output yet contributes nothing. A
// record that lists a file outside its directory, or one that cannot be
// read, is an error.
func Depend(req *plugin.Request, loader sema.Loader) ([]string, error) {
	model := req.GetModel()
	if len(model.GetImports()) == 0 || loader == nil {
		return nil, nil
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

	// A package may span several files, and a file be imported twice.
	seen := map[string]bool{}
	var read []string
	for _, imp := range model.GetImports() {
		pkg := imp.GetPackage()
		name, src, err := loader.Load(imp.GetPosition().GetFilename(), imp.GetPath())
		if err != nil || seen[name] {
			continue
		}
		seen[name] = true
		dep, err := parser.Parse(name, strings.NewReader(src))
		if err != nil {
			continue
		}

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

		dir := depOut(dep, name, req.GetTarget(), pkg)
		if dir != "" && sameDir(dir, req.GetOut()) {
			// Its record is this target's own last output.
			continue
		}
		files, paths, err := dependencyFiles(req.GetOut(), req.GetTarget(), pkg, dir)
		if err != nil {
			return nil, fmt.Errorf("reading %s's output: %w", pkg, err)
		}
		req.Dependencies = append(req.Dependencies, files...)
		read = append(read, paths...)
	}
	req.Analysis.Source.Externs = externs
	return read, nil
}

// sameDir reports whether two directories are one path.
func sameDir(a, b string) bool {
	rel, err := relative(a, b)
	return err == nil && rel == "."
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
// lists, each with its path relative to out, and returns every path it
// read: the marker, then the files.
func dependencyFiles(out, target, pkg, depOut string) ([]*plugin.DependencyFile, []string, error) {
	if depOut == "" {
		return nil, nil, nil
	}
	kept, ok, err := Recorded(depOut, target)
	if err != nil || !ok {
		return nil, nil, err
	}
	t, err := openTree(depOut)
	if err != nil {
		return nil, nil, err
	}
	defer t.Close()

	read := []string{filepath.Join(depOut, MarkerName)}
	var files []*plugin.DependencyFile
	for _, path := range kept.Paths() {
		rel, err := t.local(path, "read")
		if err != nil {
			return nil, nil, err
		}
		content, err := t.readFile(rel)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", path, err)
		}
		from, err := relative(out, path)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, &plugin.DependencyFile{Package: pkg, Path: filepath.ToSlash(from), Content: content})
		read = append(read, path)
	}
	return files, read, nil
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
