// Package reverse holds what every reverse backend's reader shares:
// diagnostics, the target block it writes, the items a roundtrip
// annotation carries as TDL, and lowering what it read.
package reverse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// Reader is the state every reader keeps.
type Reader struct {
	Diags []*plugin.Diagnostic

	// Roundtrip is set when the input carries roundtrip annotations, so
	// they are read and no directive is inferred.
	Roundtrip bool

	// Entries is the target block being written.
	Entries []*ast.TargetEntry
}

// Warn reports a loss or a construct with no TDL form.
func (r *Reader) Warn(code string, pos *ir.Position, format string, args ...any) {
	r.Diags = append(r.Diags, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
		Code:     code,
	})
}

// Directive adds `path => name(args)` to the target block, unless
// annotations carry the block.
func (r *Reader) Directive(path, name string, args ...*ast.Literal) {
	if r.Roundtrip {
		return
	}
	r.Entries = append(r.Entries, &ast.TargetEntry{Path: path, Directive: &ast.Directive{N: name, Args: args}})
}

// Pins writes a number directive on each member whose number allocation
// would not give it.
func (r *Reader) Pins(names []string, nums []int64, rule emit.NumberRule) {
	if r.Roundtrip || len(nums) == 0 {
		return
	}
	for i, pinned := range emit.Pins(nums, rule) {
		if pinned {
			r.Directive(names[i], "number", IntLit(nums[i]))
		}
	}
}

// Response is the import response for a model or the error reading it: a
// problem with the input is a diagnostic, and anything else fails the
// request.
func (r *Reader) Response(model *ir.Model, err error) (*plugin.ImportResponse, error) {
	if failed, ok := errors.AsType[*Error](err); ok {
		r.Diags = append(r.Diags, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  failed.Msg,
			Position: failed.Pos,
		})
		return &plugin.ImportResponse{Diagnostics: r.Diags}, nil
	}
	if err != nil {
		return nil, err
	}
	return &plugin.ImportResponse{Model: model, Diagnostics: r.Diags}, nil
}

// Error is a problem with the input, reported as a diagnostic.
type Error struct {
	Msg string
	Pos *ir.Position
}

func (e *Error) Error() string { return e.Msg }

// Failf returns an [Error] at a position.
func Failf(pos *ir.Position, format string, args ...any) error {
	return &Error{fmt.Sprintf(format, args...), pos}
}

// Lower lowers the file read, failing when it does not.
func Lower(file *ast.File) (*ir.Model, error) {
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		d := diags[0]
		return nil, Failf(&ir.Position{Filename: d.Pos.Filename, Line: int32(d.Pos.Line), Column: int32(d.Pos.Col)},
			"the model read from the files does not lower: %s", diags.Error())
	}
	return model, nil
}

// Item is a top-level declaration read from a file, with the index an
// annotation gave it, or -1, and the declarations reading it added.
type Item struct {
	Decl  ast.Decl
	At    int
	Extra []ast.Decl
}

// Arrange interleaves declarations read from the target with the items an
// annotation carries: each item, and each declaration given an index, at
// its index, and the rest of the declarations in the gaps, in order.
func Arrange(emitted, items []Item) []ast.Decl {
	n := len(emitted) + len(items)
	slots := make([]*Item, n)
	var rest []*Item
	place := func(it *Item) {
		if it.At >= 0 && it.At < n && slots[it.At] == nil {
			slots[it.At] = it
			return
		}
		rest = append(rest, it)
	}
	for i := range items {
		place(&items[i])
	}
	var free []*Item
	for i := range emitted {
		if emitted[i].At >= 0 {
			place(&emitted[i])
		} else {
			free = append(free, &emitted[i])
		}
	}
	free = append(free, rest...)
	var out []ast.Decl
	for _, s := range slots {
		if s == nil {
			if len(free) == 0 {
				continue
			}
			s, free = free[0], free[1:]
		}
		out = append(out, s.Decl)
		out = append(out, s.Extra...)
	}
	for _, s := range free {
		out = append(out, s.Decl)
		out = append(out, s.Extra...)
	}
	return out
}

// Head is a node's head from its doc comment and deprecation. Generating
// writes a deprecation's reason as the comment's last paragraph.
func Head(name string, doc []string, pos ast.Position, deprecated bool) ast.DeclHead {
	h := ast.DeclHead{N: name, P: pos}
	if deprecated {
		h.Dep = &ast.Deprecation{}
		if n := len(doc); n > 0 {
			if reason, ok := strings.CutPrefix(doc[n-1], "Deprecated: "); ok {
				h.Dep.Reason = reason
				doc = doc[:n-1]
				if n := len(doc); n > 0 && doc[n-1] == "" {
					doc = doc[:n-1]
				}
			}
		}
	}
	if len(doc) > 0 {
		h.Doc, h.DocP = doc, make([]ast.Position, len(doc))
	}
	return h
}

// source names the TDL an annotation carries in parse errors.
const source = "annotation.tdl"

// ParseItem parses one top-level TDL item an annotation carries.
func ParseItem(src string) (ast.Decl, error) {
	f, err := parser.Parse(source, strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	if len(f.Decls) != 1 {
		return nil, fmt.Errorf("%d declarations, want one", len(f.Decls))
	}
	return f.Decls[0], nil
}

// ParseImport parses one TDL import an annotation carries.
func ParseImport(src string) (*ast.ImportDecl, error) {
	f, err := parser.Parse(source, strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	if len(f.Imports) != 1 || len(f.Decls) != 0 {
		return nil, fmt.Errorf("want one import")
	}
	return f.Imports[0], nil
}

// ParseField parses one struct member an annotation carries.
func ParseField(src string) (*ast.Field, error) {
	f, err := parser.Parse(source, strings.NewReader("type T {\n"+src+"\n}\n"))
	if err != nil {
		return nil, err
	}
	if s, ok := f.Decls[0].(*ast.StructDecl); ok && len(s.Members) == 1 {
		if field, ok := s.Members[0].(*ast.Field); ok {
			return field, nil
		}
	}
	return nil, fmt.Errorf("want one field")
}

// ParseConforms parses a conformance list an annotation carries.
func ParseConforms(src string) ([]*ast.ClassRef, error) {
	d, err := ParseItem("type T: " + src + " {}")
	if err != nil {
		return nil, err
	}
	return d.(*ast.StructDecl).Conforms, nil
}

// StrLit is a string literal.
func StrLit(s string) *ast.Literal { return &ast.Literal{Kind: ast.LitString, Text: s} }

// IntLit is an integer literal.
func IntLit(n int64) *ast.Literal {
	return &ast.Literal{Kind: ast.LitInt, Text: strconv.FormatInt(n, 10)}
}
