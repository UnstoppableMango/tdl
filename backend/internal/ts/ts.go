// Package ts holds what the backends writing TypeScript share: the type of
// each primitive's JSON value, which names TypeScript accepts, how a
// property is quoted, and the path one generated module imports another
// by. The typescript backend writes types alone and the zod backend writes
// schemas inferring the same types, so both read these.
package ts

import (
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// DefaultFile names the file of a model without a package.
const DefaultFile = "model"

// OutDirective names where a target block's output goes; tdl reads it, and
// a dependency's reaches its import.
const OutDirective = "out"

// Scalars maps a TDL primitive to the TypeScript type of its JSON value.
// Every numeric is a number, exact only to 2^53.
var Scalars = map[string]string{
	"string":   "string",
	"int":      "number",
	"int32":    "number",
	"int64":    "number",
	"uint32":   "number",
	"uint64":   "number",
	"float32":  "number",
	"float64":  "number",
	"bool":     "boolean",
	"bytes":    "string",
	"decimal":  "string",
	"uuid":     "string",
	"instant":  "string",
	"date":     "string",
	"duration": "string",
}

// reserved is the words TypeScript refuses as a type name, `await`
// included, since an emitted file is a module.
var reserved = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		any as await boolean break case catch class const continue debugger default
		delete do else enum export extends false finally for function if
		implements import in instanceof interface let never new null number
		object package private protected public return static string super
		switch symbol this throw true try type typeof undefined unknown var
		void while with yield`) {
		reserved[w] = true
	}
}

var ident = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// Ident reports whether name is a TypeScript identifier.
func Ident(name string) bool { return ident.MatchString(name) }

// Reserved reports whether TypeScript refuses name as a type name.
func Reserved(name string) bool { return reserved[name] }

// CanName reports whether name can name a declared or imported type.
func CanName(name string) bool { return Ident(name) && !Reserved(name) }

// Property quotes a property name that is not an identifier.
func Property(name string) string {
	if Ident(name) {
		return name
	}
	return strconv.Quote(name)
}

// File is the base name, without an extension, of the module generated
// for a package.
func File(pkg string) string {
	if pkg == "" {
		return DefaultFile
	}
	return emit.LastSegment(pkg)
}

// Placed is the directory an out directive names, relative to the file
// declaring its block. The caller has checked it has an argument.
func Placed(out *ir.Directive) string {
	dir := out.GetArgs()[0].GetText()
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	return filepath.Join(filepath.Dir(out.GetPosition().GetFilename()), dir)
}

// Relative is the specifier a module generated under the out directive
// from imports the module generated for package pkg under to, ending in
// ext as Node's resolution requires: "./billing.js", or "../shop/shop.zod.js".
// It fails only when the two directories have no relative path.
func Relative(from, to *ir.Directive, pkg, ext string) (string, error) {
	rel, err := filepath.Rel(Placed(from), Placed(to))
	if err != nil {
		return "", err
	}
	spec := path.Join(filepath.ToSlash(rel), File(pkg)+ext)
	if !strings.HasPrefix(spec, "../") {
		spec = "./" + spec
	}
	return spec, nil
}
