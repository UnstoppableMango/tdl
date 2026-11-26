package salesforce

import (
	"cmp"
	"encoding/xml"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/reverse"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/lex"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
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
	// enums maps each enum a picklist reads as to its values.
	enums map[string][]string
	decls []ast.Decl
}

// objectFiles is what the input holds under one objects/<Object>/ folder.
type objectFiles struct {
	api    string
	object *plugin.File
	fields []*plugin.File
	rules  []*plugin.File
	other  []*plugin.File

	// skip is set when the object's entity name is taken, which name has
	// warned about.
	skip bool
}

// preludeNames are the prelude's declarations, which a declaration read
// would shadow.
var preludeNames = sync.OnceValue(func() map[string]bool {
	names := map[string]bool{}
	file, err := parser.Parse("prelude.tdl", strings.NewReader(prelude.Source))
	if err != nil {
		panic(err)
	}
	for _, d := range file.Decls {
		if n, ok := d.(interface{ Name() string }); ok {
			names[n.Name()] = true
		}
	}
	return names
})

// readObjects reads the object and field metadata among files into a
// model: each custom object an entity, and each picklist's values an enum.
// What has no TDL form warns and is skipped.
func readObjects(files []*plugin.File, readback bool) (*plugin.ImportResponse, error) {
	r := &reader{entities: map[string]string{}, enums: map[string][]string{}}
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
// folder first appears, and warns about each file outside one.
func (r *reader) group(files []*plugin.File) []*objectFiles {
	var out []*objectFiles
	byAPI := map[string]*objectFiles{}
	apex := false
	for _, f := range files {
		p := f.GetPath()
		api, rest, ok := objectPath(p)
		if !ok {
			if strings.HasSuffix(p, ".cls") || strings.HasSuffix(p, ".cls-meta.xml") {
				if !apex {
					// No loss code: the classes are not lost, only not read
					// yet (#1459).
					r.Warn("", filePos(f), "Apex classes are not read yet, so %s and every other class is skipped", p)
				}
				apex = true
				continue
			}
			r.Warn(emit.LossUnsupported, filePos(f), "%s is not an object or a field, and is not read", p)
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
	n := emit.Pascal(base)
	if r.taken(n) || preludeNames()[n] {
		r.Warn(emit.LossUnsupported, filePos(o.object), "%s reads as %s, which another declaration has, so the object is not read", o.api, n)
		o.skip = true
		return
	}
	r.entities[strings.ToLower(o.api)] = n
}

// skipAll warns that every file in an object folder is not read, and why.
func (r *reader) skipAll(o *objectFiles, why string) {
	for _, group := range [][]*plugin.File{{o.object}, o.fields, o.rules, o.other} {
		for _, f := range group {
			if f != nil {
				r.Warn(emit.LossUnsupported, filePos(f), "%s, so %s is not read", why, f.GetPath())
			}
		}
	}
}

// object reads one object folder into an entity, warning about everything
// in it that has no TDL form.
func (r *reader) object(o *objectFiles) error {
	base, err := custom(o.api)
	switch {
	case o.skip:
		return nil
	case err != nil:
		r.skipAll(o, err.Error())
		return nil
	case o.object == nil:
		r.skipAll(o, o.api+" has no object file")
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
	unread := r.unread(o, &obj, base)
	decl := &ast.StructDecl{DeclHead: head(name, obj.Description, pos), Keyword: "type", Conforms: []*ast.ClassRef{{N: "Entity"}}}
	r.decls = append(r.decls, decl)
	if err := r.fields(decl, o, base, unread); err != nil {
		return err
	}
	if len(unread) > 0 {
		r.Warn(emit.LossUnsupported, pos, "%s has elements with no TDL form, which are not read: %s",
			o.api, strings.Join(slices.Sorted(maps.Keys(unread)), ", "))
	}
	return nil
}

// unread returns the names of what an object holds that means something to
// an org and nothing to the model, to warn about once for the object, and
// warns about each validation rule.
func (r *reader) unread(o *objectFiles, obj *customObject, base string) map[string]bool {
	unread := map[string]bool{}
	for _, e := range obj.Other {
		if e.XMLName.Local == "validationRules" {
			r.Warn(emit.LossConstraint, at(o.object, e.XMLName.Local), "%s has a validation rule, which is not read", o.api)
			continue
		}
		unread[e.XMLName.Local] = true
	}
	for _, f := range o.rules {
		r.Warn(emit.LossConstraint, filePos(f), "%s has a validation rule, %s, which is not read", o.api, path.Base(f.GetPath()))
	}
	for _, f := range o.other {
		_, rest, _ := objectPath(f.GetPath())
		unread[strings.Split(rest, "/")[0]] = true
	}
	if obj.DeploymentStatus != "Deployed" {
		unread["deploymentStatus"] = true
	}
	if obj.SharingModel != "ReadWrite" {
		unread["sharingModel"] = true
	}
	if obj.NameField != (nameField{DisplayFormat: base + "-{0000}", Label: obj.Label + " Number", Type: "AutoNumber"}) {
		unread["nameField"] = true
	}
	return unread
}

// fields reads an object's field files into decl, and its external ID into
// a key directive.
func (r *reader) fields(decl *ast.StructDecl, o *objectFiles, base string, unread map[string]bool) error {
	name := decl.N
	var key string
	seen := map[string]bool{}
	for _, f := range o.fields {
		before := len(r.Entries)
		fld, isKey, err := r.field(name, o.api, base, f, unread)
		if err != nil {
			return err
		}
		if fld == nil {
			continue
		}
		if seen[strings.ToLower(fld.N)] {
			r.Entries = r.Entries[:before]
			r.Warn(emit.LossUnsupported, filePos(f), "%s reads as %s.%s, which another field has, so it is not read", path.Base(f.GetPath()), name, fld.N)
			continue
		}
		seen[strings.ToLower(fld.N)] = true
		decl.Members = append(decl.Members, fld)
		switch {
		case isKey && key == "":
			key = fld.N
			r.Directive(name, "key", &ast.Literal{Kind: ast.LitName, Text: fld.N})
		case isKey:
			r.Warn(emit.LossUnsupported, filePos(f), "%s has a second external ID, %s, and only %s is read as its key", o.api, fld.N, key)
		}
	}
	return nil
}

// field reads one field file, returning nil for a field it skips, and
// whether the field is an external ID. Elements with no model meaning are
// added to unread.
func (r *reader) field(entity, api, object string, f *plugin.File, unread map[string]bool) (*ast.Field, bool, error) {
	var cf customField
	if err := xml.Unmarshal(f.GetContent(), &cf); err != nil {
		return nil, false, reverse.Failf(filePos(f), "%s is not field metadata: %v", f.GetPath(), err)
	}
	pos := filePos(f)
	base, err := custom(cf.FullName)
	if err != nil {
		r.Warn(emit.LossUnsupported, pos, "%s, so field %s.%s is not read", err, api, cf.FullName)
		return nil, false, nil
	}
	name := emit.Camel(base)
	what := api + "." + cf.FullName
	// The field's own directives go before those its type writes.
	before := len(r.Entries)
	typ := r.fieldType(&cf, entity, object, base, what, pos, unread)
	if typ == nil {
		r.Entries = r.Entries[:before]
		return nil, false, nil
	}
	typed := slices.Clone(r.Entries[before:])
	r.Entries = r.Entries[:before]
	if !cf.Required && cf.Type != "Checkbox" && cf.Type != "MultiselectPicklist" {
		typ.Optional = true
	}

	path := entity + "." + name
	if emit.Pascal(name) != base {
		r.Directive(path, "name", reverse.StrLit(base))
	}
	if cf.Label != label(base) {
		r.Directive(path, "label", reverse.StrLit(cf.Label))
	}
	r.Entries = append(r.Entries, typed...)
	for _, e := range cf.Other {
		unread[e.XMLName.Local] = true
	}
	if cf.Unique && !cf.ExternalID {
		r.Warn(emit.LossConstraint, at(f, "unique"), "%s is unique, which is not read", what)
	}
	key := cf.ExternalID
	if key && cf.Type != "Text" && cf.Type != "Number" {
		r.Warn(emit.LossUnsupported, pos, "%s is an external ID, which a %s field regenerates without, so it is not read as a key", what, cf.Type)
		key = false
	} else if key && !cf.Unique {
		unread["unique"] = true
	}
	return &ast.Field{DeclHead: head(name, cf.Description, pos), Type: typ}, key, nil
}

// fieldType is the type a field reads as, or nil when it has none.
func (r *reader) fieldType(cf *customField, entity, object, base, what string, pos *ir.Position, unread map[string]bool) *ast.TypeRef {
	scale := 0
	if cf.Scale != nil {
		scale = *cf.Scale
	}
	path := entity + "." + emit.Camel(base)
	switch cf.Type {
	case "Text":
		switch {
		case cf.Length < 1 || cf.Length > columns["string"].length:
			r.Warn(emit.LossUnsupported, pos, "%s holds %d characters, which regenerates as %d", what, cf.Length, columns["string"].length)
		case cf.Length != columns["string"].length:
			r.Directive(path, "length", reverse.IntLit(int64(cf.Length)))
		}
		return &ast.TypeRef{N: "string"}
	case "Number":
		if cf.Precision != columns["decimal"].precision {
			r.Warn(emit.LossUnsupported, pos, "%s holds %d digits, which regenerates as %d", what, cf.Precision, columns["decimal"].precision)
		}
		if scale == 0 {
			return &ast.TypeRef{N: "int64"}
		}
		switch {
		case scale < 0 || scale > columns["decimal"].precision:
			r.Warn(emit.LossUnsupported, pos, "%s has %d decimal places, which regenerates as %d", what, scale, columns["decimal"].scale)
		case scale != columns["decimal"].scale:
			r.Directive(path, "scale", reverse.IntLit(int64(scale)))
		}
		return &ast.TypeRef{N: "decimal"}
	case "Checkbox":
		if cf.Required {
			r.Respell(emit.LossOptional, pos, "%s is required, and a checkbox, which is never empty, regenerates without saying so", what)
		}
		if cf.DefaultValue != "false" {
			r.Warn(emit.LossDefault, pos, "%s defaults to %s, which is not read", what, cmp.Or(cf.DefaultValue, "nothing"))
		}
		return &ast.TypeRef{N: "bool"}
	case "Date":
		return &ast.TypeRef{N: "date"}
	case "DateTime":
		return &ast.TypeRef{N: "instant"}
	case "Lookup":
		target, ok := r.entities[strings.ToLower(cf.ReferenceTo)]
		if !ok {
			why := "is not among the objects imported"
			if _, err := custom(cf.ReferenceTo); err != nil {
				why = "is a standard or managed-package object"
			}
			r.Warn(emit.LossUnsupported, pos, "%s looks up %s, which %s, so the field is not read", what, cf.ReferenceTo, why)
			return nil
		}
		want := "SetNull"
		if cf.Required {
			want = "Restrict"
		}
		if cf.DeleteConstraint != want {
			unread["deleteConstraint"] = true
		}
		if cf.RelationshipName != object+base || cf.RelationshipLabel != label(object)+" "+cf.Label {
			unread["relationshipName"] = true
		}
		return &ast.TypeRef{N: target}
	case "Picklist", "MultiselectPicklist":
		enum := r.picklist(cf, entity, base, what, pos, unread)
		if enum == "" {
			return nil
		}
		if cf.Type == "Picklist" {
			return &ast.TypeRef{N: enum}
		}
		if cf.Required {
			r.Respell(emit.LossOptional, pos, "%s is required, and a multi-select picklist regenerates without saying so", what)
		}
		if cf.VisibleLines != 4 {
			unread["visibleLines"] = true
		}
		return &ast.TypeRef{Set: &ast.TypeRef{N: enum}}
	}
	r.Warn(emit.LossUnsupported, pos, "%s is a %s field, which is not read", what, cmp.Or(cf.Type, "typeless"))
	return nil
}

// picklist returns the enum a picklist's values read as, declaring it the
// first time, or "" when they cannot be one. The enum is named for the
// field, and for its entity too when another enum has the name.
func (r *reader) picklist(cf *customField, entity, base, what string, pos *ir.Position, unread map[string]bool) string {
	if cf.ValueSet == nil || len(cf.ValueSet.Definition.Values) == 0 {
		r.Warn(emit.LossUnsupported, pos, "%s has no values of its own, so the field is not read", what)
		return ""
	}
	set := cf.ValueSet
	if !set.Restricted || set.Definition.Sorted {
		unread["valueSet"] = true
	}
	var values []string
	for _, v := range set.Definition.Values {
		if _, keyword := lex.Lookup(v.FullName); keyword || !identifier(v.FullName) {
			r.Warn(emit.LossUnsupported, pos, "%s has the value %q, which is not a TDL name, so the field is not read", what, v.FullName)
			return ""
		}
		if v.Label != label(v.FullName) {
			unread["valueSet"] = true
		}
		if v.Default {
			r.Warn(emit.LossDefault, pos, "%s defaults to %s, which is not read", what, v.FullName)
		}
		values = append(values, v.FullName)
	}

	name := emit.Pascal(base)
	if have, ok := r.enums[name]; ok && !slices.Equal(have, values) || r.taken(name) || preludeNames()[name] {
		name = entity + name
	}
	if r.taken(name) || preludeNames()[name] {
		r.Warn(emit.LossUnsupported, pos, "%s reads as enum %s, which another declaration has, so the field is not read", what, name)
		return ""
	}
	if have, ok := r.enums[name]; ok {
		if slices.Equal(have, values) {
			return name
		}
		r.Warn(emit.LossUnsupported, pos, "%s reads as enum %s, which another field declares with other values, so the field is not read", what, name)
		return ""
	}
	r.enums[name] = values
	decl := &ast.EnumDecl{DeclHead: ast.DeclHead{N: name, P: astPos(pos)}}
	for _, v := range values {
		decl.Variants = append(decl.Variants, &ast.Variant{DeclHead: ast.DeclHead{N: v, P: astPos(pos)}})
	}
	r.decls = append(r.decls, decl)
	return name
}

// taken reports whether an entity is named n.
func (r *reader) taken(n string) bool {
	return slices.Contains(slices.Collect(maps.Values(r.entities)), n)
}

func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		letter := c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
		if !letter && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return true
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

// at is where an element first opens in a file, or the file's start.
func at(f *plugin.File, elem string) *ir.Position {
	src := string(f.GetContent())
	i := strings.Index(src, "<"+elem+">")
	if i < 0 {
		return filePos(f)
	}
	line := strings.Count(src[:i], "\n")
	col := i - strings.LastIndex(src[:i], "\n")
	return &ir.Position{Filename: f.GetPath(), Line: int32(line + 1), Column: int32(col)}
}

func astPos(p *ir.Position) ast.Position {
	return ast.Position{Filename: p.GetFilename(), Line: int(p.GetLine()), Col: int(p.GetColumn())}
}
