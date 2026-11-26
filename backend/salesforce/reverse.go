package salesforce

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// importPackage is the package an import reads into, since Salesforce
// source names none.
const importPackage = "salesforce"

// reader reads custom objects and their fields into entities.
type reader struct {
	reverse.Reader

	// entities maps each custom object read, lower-cased since Salesforce
	// ignores case, to the entity it reads as.
	entities map[string]string
	decls    []ast.Decl
}

// objectFiles is what the input holds under one objects/<Object>/ folder.
type objectFiles struct {
	api    string
	object *plugin.File
	fields []*plugin.File
	rules  []*plugin.File
	other  []*plugin.File
}

// readObjects reads the object and field metadata among files into a
// model: each custom object an entity.
// What has no TDL form warns and is skipped.
func readObjects(files []*plugin.File, readback bool) (*plugin.ImportResponse, error) {
	r := &reader{entities: map[string]string{}}
	r.Readback = readback
	return r.Response(r.read(files))
}

func (r *reader) read(files []*plugin.File) (*ir.Model, error) {
	objects := r.group(files)
	for _, o := range objects {
		r.name(o)
	}
	for _, o := range objects {
		if err := r.object(o); err != nil {
			return nil, err
		}
	}
	file := &ast.File{Package: &ast.PackageDecl{Path: importPackage}, Decls: r.decls}
	if len(r.Entries) > 0 {
		file.Decls = append(file.Decls, &ast.TargetDecl{DeclHead: ast.DeclHead{N: Name}, For: importPackage, Entries: r.Entries})
	}
	return reverse.Lower(file)
}

// group sorts files by the object folder holding them, in the order each
// folder first appears.
func (r *reader) group(files []*plugin.File) []*objectFiles {
	var out []*objectFiles
	byAPI := map[string]*objectFiles{}
	for _, f := range files {
		p := f.GetPath()
		api, rest, ok := objectPath(p)
		if !ok {
			continue
		}
		o := byAPI[api]
		if o == nil {
			o = &objectFiles{api: api}
			byAPI[api] = o
			out = append(out, o)
		}
		switch {
		case rest == api+".object-meta.xml":
			o.object = f
		case strings.HasPrefix(rest, "fields/") && strings.HasSuffix(rest, ".field-meta.xml"):
			o.fields = append(o.fields, f)
		case strings.HasPrefix(rest, "validationRules/"):
			o.rules = append(o.rules, f)
		default:
			o.other = append(o.other, f)
		}
	}
	return out
}

// objectPath splits a path at its last objects/<Object>/ segment, so a
// folder above the project named objects does not take its place.
func objectPath(p string) (api, rest string, ok bool) {
	parts := strings.Split(p, "/")
	for i := len(parts) - 3; i >= 0; i-- {
		if parts[i] == "objects" {
			return parts[i+1], strings.Join(parts[i+2:], "/"), true
		}
	}
	return "", "", false
}

// custom returns a custom object's or field's name without __c, or says
// why it is not one import reads.
func custom(api string) (string, error) {
	base, ok := strings.CutSuffix(api, "__c")
	switch {
	case !ok:
		return "", fmt.Errorf("%s is a standard name", api)
	case strings.Contains(base, "__"):
		return "", fmt.Errorf("%s is in a managed package's namespace", api)
	}
	return base, nil
}

// name records the entity a custom object reads as, so a lookup to it
// resolves whatever order the objects come in.
func (r *reader) name(o *objectFiles) {
	base, err := custom(o.api)
	if err != nil || o.object == nil {
		return
	}
	r.entities[strings.ToLower(o.api)] = emit.Pascal(base)
}

// object reads one object folder into an entity, warning about everything
// in it that has no TDL form.
func (r *reader) object(o *objectFiles) error {
	base, err := custom(o.api)
	if err != nil || o.object == nil {
		// What is skipped warns in the next part of the stack.
		return nil
	}
	var obj customObject
	if err := xml.Unmarshal(o.object.GetContent(), &obj); err != nil {
		return reverse.Failf(filePos(o.object), "%s is not object metadata: %v", o.object.GetPath(), err)
	}
	name := r.entities[strings.ToLower(o.api)]
	if emit.Pascal(name) != base {
		r.Directive(name, "name", reverse.StrLit(base))
	}
	lbl := label(base)
	if obj.Label != lbl {
		r.Directive(name, "label", reverse.StrLit(obj.Label))
	}
	if obj.PluralLabel != obj.Label+"s" {
		r.Directive(name, "plural", reverse.StrLit(obj.PluralLabel))
	}

	pos := filePos(o.object)
	decl := &ast.StructDecl{DeclHead: head(name, obj.Description, pos), Keyword: "type", Conforms: []*ast.ClassRef{{N: "Entity"}}}
	r.decls = append(r.decls, decl)
	return nil
}

// head is a node's head from its description, whose last line is a
// deprecation when generating wrote one.
func head(name, desc string, pos *ir.Position) ast.DeclHead {
	h := ast.DeclHead{N: name, P: astPos(pos)}
	if desc == "" {
		return h
	}
	doc := strings.Split(desc, "\n")
	if reason, ok := strings.CutPrefix(doc[len(doc)-1], "Deprecated."); ok {
		h.Dep = &ast.Deprecation{Reason: strings.TrimSpace(reason)}
		doc = doc[:len(doc)-1]
	}
	if len(doc) > 0 {
		h.Doc, h.DocP = doc, make([]ast.Position, len(doc))
	}
	return h
}

func filePos(f *plugin.File) *ir.Position {
	return &ir.Position{Filename: f.GetPath(), Line: 1, Column: 1}
}

func astPos(p *ir.Position) ast.Position {
	return ast.Position{Filename: p.GetFilename(), Line: int(p.GetLine()), Col: int(p.GetColumn())}
}
