package golang

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A bare `json` directive on the target block opts in to the JSON wire
// convention the schema backends share (backend/internal/emit/wire.go): a
// property is named as the model spells it, and a fielded enum is tagged
// internally by its discriminant.
//
// Every field gets a `json` struct tag, unless a `tag` directive sets its
// tag. A fielded enum's variants marshal with the discriminant, and an
// Unmarshal<Enum> function decodes one by it. encoding/json cannot decode
// an interface, so a struct holding an enum, directly or in a collection,
// gets an UnmarshalJSON that decodes those fields with it. A type holding a
// list or map gets a MarshalJSON writing an empty one as [] or {}, never
// null (see fill).

// jsonLocals are the identifiers the generated JSON methods declare, which
// a type parameter would shadow.
var jsonLocals = []string{"plain", "data", "raw", "err", "v"}

// jsonTag is a field's struct tag under the convention: its name in the
// model, omitted when the field is optional and absent.
func (g *generator) jsonTag(f *ir.Field) string {
	name := f.GetMeta().GetName()
	if g.optional(f.GetType()) {
		name += ",omitempty"
	}
	return `json:` + strconv.Quote(name)
}

// optional reports whether a type is Option, past aliases.
func (g *generator) optional(id *ir.ID) bool {
	t := g.Model.Type(id)
	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return false
	}
	if a := decl.GetAlias(); a != nil && len(t.GetArgs()) == 0 {
		return g.optional(a.GetTarget())
	}
	return decl.GetEnumeration() != nil && decl.GetMeta().GetName() == "Option" && len(t.GetArgs()) == 1
}

// wireName is the JSON property encoding/json reads a field from, or ""
// when its tag hides it.
func (g *generator) wireName(f *ir.Field) string {
	tag, ok := g.Text(f.GetDirectives(), "tag")
	if !ok {
		return f.GetMeta().GetName()
	}
	j, ok := reflect.StructTag(tag).Lookup("json")
	if !ok {
		return g.fieldName(f)
	}
	name, _, _ := strings.Cut(j, ",")
	switch name {
	case "-":
		return ""
	case "":
		return g.fieldName(f)
	}
	return name
}

// jsonProblem says why a declaration's JSON methods cannot be generated.
func (g *generator) jsonProblem(decl *ir.Decl) error {
	for _, p := range decl.Params() {
		if slices.Contains(jsonLocals, p.GetName()) {
			return emit.Unsupported(decl.GetMeta().GetPosition(),
				"%s has a type parameter named %s, which its JSON methods declare", decl.GetMeta().GetName(), p.GetName())
		}
	}
	return nil
}

// writeUnmarshal writes an UnmarshalJSON for a struct holding a field
// encoding/json cannot decode by itself: a fielded enum, or a collection or
// option of one. Every other field is decoded as usual.
func (g *generator) writeUnmarshal(b *strings.Builder, decl *ir.Decl, name string, fields []*ir.Field) error {
	type item struct{ field, wire, dec string }
	var items []item
	for _, f := range fields {
		wire := g.wireName(f)
		if wire == "" {
			continue
		}
		dec, ok, err := g.decoder(f.GetType(), nil, f.GetMeta().GetPosition())
		if err != nil {
			return err
		}
		if ok {
			items = append(items, item{g.fieldName(f), wire, dec})
		}
	}
	if len(items) == 0 {
		return nil
	}
	if err := g.jsonProblem(decl); err != nil {
		return err
	}
	g.use("encoding/json")
	g.use("fmt")

	recv, self := receiver(decl, name), name+typeArgs(decl)
	fmt.Fprintf(b, "\n// UnmarshalJSON decodes a %s, choosing the variant of each enum it holds by its discriminant.\n", name)
	fmt.Fprintf(b, "func (%s *%s) UnmarshalJSON(data []byte) error {\n", recv, self)
	fmt.Fprintf(b, "type plain %s\nvar raw struct {\n*plain\n", self)
	for _, it := range items {
		fmt.Fprintf(b, "%s json.RawMessage `json:%s`\n", it.field, strconv.Quote(it.wire))
	}
	fmt.Fprintf(b, "}\nraw.plain = (*plain)(%s)\n", recv)
	b.WriteString("if err := json.Unmarshal(data, &raw); err != nil {\nreturn err\n}\n")
	for _, it := range items {
		fmt.Fprintf(b, "if raw.%s != nil && string(raw.%s) != \"null\" {\n", it.field, it.field)
		fmt.Fprintf(b, "v, err := %s(raw.%s)\n", it.dec, it.field)
		fmt.Fprintf(b, "if err != nil {\nreturn fmt.Errorf(%s, err)\n}\n", strconv.Quote(it.wire+": %w"))
		fmt.Fprintf(b, "%s.%s = v\n}\n", recv, it.field)
	}
	b.WriteString("return nil\n}\n")
	return nil
}

// writeEnumJSON writes a fielded enum's JSON methods: a MarshalJSON on each
// variant writing the discriminant first, and Unmarshal<Enum> choosing the
// variant by it.
func (g *generator) writeEnumJSON(b *strings.Builder, decl *ir.Decl, name string) error {
	if err := g.jsonProblem(decl); err != nil {
		return err
	}
	disc := g.Discriminant(decl)
	for _, v := range decl.GetEnumeration().GetVariants() {
		for _, f := range v.GetFields() {
			if g.wireName(f) == disc {
				return emit.Unsupported(f.GetMeta().GetPosition(),
					"%s.%s.%s takes the name of the discriminant, and a discriminant directive can rename it",
					decl.GetMeta().GetName(), v.GetMeta().GetName(), f.GetMeta().GetName())
			}
		}
	}
	fn := "Unmarshal" + name
	if g.declares(fn) {
		return emit.Unsupported(decl.GetMeta().GetPosition(),
			"%s would be decoded by %s, which a declaration is named", decl.GetMeta().GetName(), fn)
	}
	g.use("encoding/json")
	g.use("errors")
	g.use("fmt")

	args := typeArgs(decl)
	for _, v := range decl.GetEnumeration().GetVariants() {
		variant := g.variantName(name, v) + args
		recv := receiver(decl, g.variantName(name, v))
		var fills strings.Builder
		filled, err := g.fillFields(&fills, v.GetFields())
		if err != nil {
			return err
		}
		value, doc := "plain("+recv+")", ""
		if filled {
			if err := g.marshalProblem(decl); err != nil {
				return err
			}
			if err := g.marshalCollides(decl, g.variantName(name, v), v.GetFields()); err != nil {
				return err
			}
			value, doc = "self", strings.TrimSuffix(fillDoc, ".")
		}
		fmt.Fprintf(b, "\n// MarshalJSON encodes a %s with its %s%s.\n", variant, strconv.Quote(disc), doc)
		fmt.Fprintf(b, "func (%s %s) MarshalJSON() ([]byte, error) {\n", recv, variant)
		fmt.Fprintf(b, "type plain %s\n", variant)
		if filled {
			fmt.Fprintf(b, "self := plain(%s)\n%s", recv, fills.String())
		}
		fmt.Fprintf(b, "return json.Marshal(struct {\nTDLDiscriminant string `json:%s`\nplain\n}{%s, %s})\n}\n",
			strconv.Quote(disc), strconv.Quote(emit.Tag(v)), value)
	}

	self := name + args
	fmt.Fprintf(b, "\n// %s decodes a %s, choosing the variant by its %s.\n", fn, name, strconv.Quote(disc))
	fmt.Fprintf(b, "func %s%s(data []byte) (%s, error) {\n", fn, g.typeParams(decl), self)
	fmt.Fprintf(b, "var head struct {\nTDLDiscriminant *string `json:%s`\n}\n", strconv.Quote(disc))
	b.WriteString("if err := json.Unmarshal(data, &head); err != nil {\nreturn nil, err\n}\n")
	fmt.Fprintf(b, "if head.TDLDiscriminant == nil {\nreturn nil, errors.New(%s)\n}\n",
		strconv.Quote(name+": no "+strconv.Quote(disc)+" property"))
	b.WriteString("switch *head.TDLDiscriminant {\n")
	for _, v := range decl.GetEnumeration().GetVariants() {
		fmt.Fprintf(b, "case %s:\nvar v %s\nerr := json.Unmarshal(data, &v)\nreturn v, err\n",
			strconv.Quote(emit.Tag(v)), g.variantName(name, v)+args)
	}
	fmt.Fprintf(b, "}\nreturn nil, fmt.Errorf(%s, *head.TDLDiscriminant)\n}\n",
		strconv.Quote(name+": unknown "+disc+" %q"))
	return nil
}

// decoder is a Go expression decoding the JSON of a type that encoding/json
// cannot decode by itself, a func([]byte) (T, error), and whether one is
// needed. A struct decodes itself, so only enums, and the collections,
// options, and newtypes holding them, need one.
func (g *generator) decoder(id *ir.ID, fr *frame, at *ir.Position) (string, bool, error) {
	t := g.Model.Type(id)
	if t == nil || t.GetExtern() != nil || t.GetUnit() != nil {
		return "", false, nil
	}
	if ref := t.GetParam(); ref != nil {
		if fr == nil || int(ref.GetIndex()) >= len(fr.args) {
			return "", false, nil
		}
		return g.decoder(fr.args[ref.GetIndex()], fr.outer, at)
	}
	decl := g.Model.Decl(t.GetCtor())
	if decl == nil || g.isForeign(decl) {
		return "", false, nil
	}
	name, args := decl.GetMeta().GetName(), t.GetArgs()
	if a := decl.GetAlias(); a != nil {
		return g.decoder(a.GetTarget(), bind(args, fr), at)
	}

	goType := func() (string, error) { return g.typeIn(id, fr) }
	elem := func(i int) (string, string, bool, error) {
		if i >= len(args) {
			return "", "", false, nil
		}
		dec, ok, err := g.decoder(args[i], fr, at)
		if err != nil || !ok {
			return "", "", false, err
		}
		typ, err := g.typeIn(args[i], fr)
		return dec, typ, err == nil, err
	}

	switch {
	case decl.GetPrimitive() != nil:
		if name == "Set" && g.wire {
			g.Warn(emit.Unsupported(at,
				"a Set is a Go map, which encoding/json writes as an object and not the array the JSON wire convention holds"))
		}
		switch name {
		case "List", "Set":
			dec, _, ok, err := elem(0)
			if !ok {
				return "", false, err
			}
			out, err := goType()
			if err != nil {
				return "", false, err
			}
			put := "out[i] = v"
			if name == "Set" {
				put = "out[v] = struct{}{}"
			}
			return fmt.Sprintf(`func(data []byte) (%s, error) {
var raws []json.RawMessage
if err := json.Unmarshal(data, &raws); err != nil || raws == nil {
return nil, err
}
out := make(%s, len(raws))
for i, raw := range raws {
v, err := %s(raw)
if err != nil {
return nil, fmt.Errorf("[%%d]: %%w", i, err)
}
%s
}
return out, nil
}`, out, out, dec, put), true, nil
		case "Map":
			dec, _, ok, err := elem(1)
			if !ok {
				return "", false, err
			}
			key, err := g.typeIn(args[0], fr)
			if err != nil {
				return "", false, err
			}
			out, err := goType()
			if err != nil {
				return "", false, err
			}
			return fmt.Sprintf(`func(data []byte) (%s, error) {
var raws map[%s]json.RawMessage
if err := json.Unmarshal(data, &raws); err != nil || raws == nil {
return nil, err
}
out := make(%s, len(raws))
for k, raw := range raws {
v, err := %s(raw)
if err != nil {
return nil, fmt.Errorf("[%%v]: %%w", k, err)
}
out[k] = v
}
return out, nil
}`, out, key, out, dec), true, nil
		}
		return "", false, nil

	case decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1:
		dec, typ, ok, err := elem(0)
		if !ok {
			return "", false, err
		}
		return fmt.Sprintf(`func(data []byte) (*%s, error) {
if string(data) == "null" {
return nil, nil
}
v, err := %s(data)
if err != nil {
return nil, err
}
return &v, nil
}`, typ, dec), true, nil

	case decl.GetNewtype() != nil:
		dec, ok, err := g.decoder(decl.GetNewtype().GetBase(), bind(args, fr), at)
		if !ok {
			return "", false, err
		}
		out, err := goType()
		if err != nil {
			return "", false, err
		}
		return fmt.Sprintf(`func(data []byte) (%s, error) {
v, err := %s(data)
return %s(v), err
}`, out, dec, out), true, nil
	}

	// A generic struct or enum decodes its parameters with encoding/json,
	// which cannot decode an enum standing for one.
	for _, a := range args {
		if _, ok, _ := g.decoder(a, fr, at); ok {
			g.Warn(emit.Unsupported(at,
				"%s is applied to a type that encoding/json cannot decode, and its JSON methods decode its parameters as plain values", name))
			break
		}
	}

	if e := decl.GetEnumeration(); e != nil && emit.Fielded(e) && emit.IsOwn(decl) {
		out, err := goType()
		if err != nil {
			return "", false, err
		}
		// Unmarshal<Enum>, instantiated as the type is.
		return "Unmarshal" + out, true, nil
	}
	return "", false, nil
}

// encoding/json writes a nil slice or map as null, and the convention writes
// an empty collection as [] or {} at every depth. A struct, variant, or
// newtype holding a list or map, directly or through an option, alias, or
// another list or map, gets a MarshalJSON that marshals a copy with every
// nil one replaced by an empty one; a declaration marshalling itself is left
// to its own MarshalJSON. A Set is a Go map, which already warns, and is
// left alone.

// fillDoc ends the doc comment of every MarshalJSON that fills, which
// import reads as this backend's output.
const fillDoc = ", writing each empty list as [] and each empty map as {}, never null."

// marshalLocals are the identifiers a generated MarshalJSON declares, past
// the numbered ones fillLocal names.
var marshalLocals = []string{"plain", "self"}

// marshalProblem says why a declaration's MarshalJSON cannot be generated.
func (g *generator) marshalProblem(decl *ir.Decl) error {
	for _, p := range decl.Params() {
		name := p.GetName()
		if slices.Contains(marshalLocals, name) || fillLocal(name) {
			return emit.Unsupported(decl.GetMeta().GetPosition(),
				"%s has a type parameter named %s, which its MarshalJSON declares", decl.GetMeta().GetName(), name)
		}
	}
	return nil
}

// marshalCollides refuses a type with a field Go names MarshalJSON, which
// the method would collide with.
func (g *generator) marshalCollides(decl *ir.Decl, name string, fields []*ir.Field) error {
	for _, f := range fields {
		if g.fieldName(f) == "MarshalJSON" {
			return emit.Unsupported(decl.GetMeta().GetPosition(),
				"%s has a field named MarshalJSON, which the MarshalJSON writing its empty collections would collide with", name)
		}
	}
	return nil
}

// fillLocal reports whether a name is one fill declares: out, e, or k and a
// depth.
func fillLocal(name string) bool {
	for _, prefix := range []string{"out", "e", "k"} {
		if rest, ok := strings.CutPrefix(name, prefix); ok && rest != "" && strings.Trim(rest, "0123456789") == "" {
			return true
		}
	}
	return false
}

// needsFill reports whether a value of a type holds a list or map that its own
// MarshalJSON, if any, does not fill: a list or map, or an option, alias, or
// newtype over a pointer holding one.
func (g *generator) needsFill(id *ir.ID, fr *frame) bool {
	t, decl, fr, ok := g.resolveFill(id, fr)
	if !ok {
		return false
	}
	args := t.GetArgs()
	switch {
	case decl.GetPrimitive() != nil:
		name := decl.GetMeta().GetName()
		return (name == "List" || name == "Map") && len(args) > 0
	case pointer(decl, args):
		return g.needsFill(args[0], fr)
	case decl.GetNewtype() != nil:
		return !g.marshals(decl) && g.pointerNewtype(decl) && g.needsFill(decl.GetNewtype().GetBase(), bind(args, fr))
	}
	return false
}

// resolveFill is the declaration a type names past aliases and type
// parameters, with the frame its arguments are read in, or false for a type
// fill leaves alone: an extern, a unit, a foreign declaration, or a
// parameter no frame binds.
func (g *generator) resolveFill(id *ir.ID, fr *frame) (*ir.Type, *ir.Decl, *frame, bool) {
	for {
		t := g.Model.Type(id)
		if t == nil || t.GetExtern() != nil || t.GetUnit() != nil {
			return nil, nil, nil, false
		}
		if ref := t.GetParam(); ref != nil {
			if fr == nil || int(ref.GetIndex()) >= len(fr.args) {
				return nil, nil, nil, false
			}
			id, fr = fr.args[ref.GetIndex()], fr.outer
			continue
		}
		decl := g.Model.Decl(t.GetCtor())
		if decl == nil || g.isForeign(decl) {
			return nil, nil, nil, false
		}
		if a := decl.GetAlias(); a != nil {
			id, fr = a.GetTarget(), bind(t.GetArgs(), fr)
			continue
		}
		return t, decl, fr, true
	}
}

// pointer reports whether a declaration applied to args is Option or
// Nullable, a Go pointer.
func pointer(decl *ir.Decl, args []*ir.ID) bool {
	name := decl.GetMeta().GetName()
	return decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1
}

// pointerNewtype reports whether a newtype is a Go pointer type, which
// cannot declare a method.
func (g *generator) pointerNewtype(decl *ir.Decl) bool {
	for seen := map[*ir.Decl]bool{}; decl.GetNewtype() != nil && !seen[decl]; {
		seen[decl] = true
		t, base, _, ok := g.resolveFill(decl.GetNewtype().GetBase(), nil)
		if !ok {
			return false
		}
		if pointer(base, t.GetArgs()) {
			return true
		}
		decl = base
	}
	return false
}

// marshals reports whether a struct or newtype gets a MarshalJSON filling
// its collections.
func (g *generator) marshals(decl *ir.Decl) bool {
	if v, ok := g.marshalled[decl]; ok {
		return v
	}
	if g.marshalled == nil {
		g.marshalled = map[*ir.Decl]bool{}
	}
	// A declaration reaching itself adds nothing to what it already marshals.
	g.marshalled[decl] = false
	v := g.computeMarshals(decl)
	g.marshalled[decl] = v
	return v
}

// computeMarshals is marshals without the memo: a struct with a field to
// fill, or a newtype that is not a pointer, over a collection to fill or a
// declaration with its own MarshalJSON.
func (g *generator) computeMarshals(decl *ir.Decl) bool {
	if !g.wire || !emit.IsOwn(decl) || g.isForeign(decl) {
		return false
	}
	if decl.GetStructure() != nil {
		return slices.ContainsFunc(decl.Fields(), g.fillsField)
	}
	nt := decl.GetNewtype()
	if nt == nil || g.pointerNewtype(decl) {
		return false
	}
	if g.needsFill(nt.GetBase(), nil) {
		return true
	}
	// A newtype over a struct or newtype marshalling itself loses its
	// method, and converts back to call it.
	_, base, _, ok := g.resolveFill(nt.GetBase(), nil)
	return ok && (base.GetStructure() != nil || base.GetNewtype() != nil) && g.marshals(base)
}

// fillsField reports whether a field encoding/json writes holds a
// collection to fill.
func (g *generator) fillsField(f *ir.Field) bool {
	return g.wireName(f) != "" && g.needsFill(f.GetType(), nil)
}

// fill writes statements replacing each nil list or map in lhs, a value of
// the type, with an empty one. What it changes it copies first, so the value
// being marshalled is left as it was.
func (g *generator) fill(b *strings.Builder, lhs string, id *ir.ID, fr *frame, depth int) error {
	if !g.needsFill(id, fr) {
		return nil
	}
	t, decl, fr, _ := g.resolveFill(id, fr)
	args := t.GetArgs()
	if decl.GetNewtype() != nil {
		// A pointer newtype dereferences as its base does.
		return g.fill(b, lhs, decl.GetNewtype().GetBase(), bind(args, fr), depth)
	}
	e := fmt.Sprintf("e%d", depth)
	if pointer(decl, args) {
		fmt.Fprintf(b, "if %s != nil {\n%s := *%s\n", lhs, e, lhs)
		if err := g.fill(b, e, args[0], fr, depth+1); err != nil {
			return err
		}
		fmt.Fprintf(b, "%s = &%s\n}\n", lhs, e)
		return nil
	}

	typ, err := g.typeIn(id, fr)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "if %s == nil {\n%s = %s{}\n}", lhs, lhs, typ)
	elem := args[0]
	if decl.GetMeta().GetName() == "Map" {
		if len(args) < 2 {
			return nil
		}
		elem = args[1]
	}
	if !g.needsFill(elem, fr) {
		b.WriteString("\n")
		return nil
	}
	out := fmt.Sprintf("out%d", depth)
	if decl.GetMeta().GetName() == "Map" {
		fmt.Fprintf(b, " else {\n%s := make(%s, len(%s))\n", out, typ, lhs)
		k := fmt.Sprintf("k%d", depth)
		fmt.Fprintf(b, "for %s, %s := range %s {\n", k, e, lhs)
		if err := g.fill(b, e, elem, fr, depth+1); err != nil {
			return err
		}
		fmt.Fprintf(b, "%s[%s] = %s\n}\n", out, k, e)
	} else {
		fmt.Fprintf(b, " else {\n%s := make(%s, 0, len(%s))\n", out, typ, lhs)
		fmt.Fprintf(b, "for _, %s := range %s {\n", e, lhs)
		if err := g.fill(b, e, elem, fr, depth+1); err != nil {
			return err
		}
		fmt.Fprintf(b, "%s = append(%s, %s)\n}\n", out, out, e)
	}
	fmt.Fprintf(b, "%s = %s\n}\n", lhs, out)
	return nil
}

// fillFields writes fill for each field of self that holds a collection,
// and reports whether any did.
func (g *generator) fillFields(b *strings.Builder, fields []*ir.Field) (bool, error) {
	filled := false
	for _, f := range fields {
		if !g.fillsField(f) {
			continue
		}
		filled = true
		if err := g.fill(b, "self."+g.fieldName(f), f.GetType(), nil, 0); err != nil {
			return false, err
		}
	}
	return filled, nil
}

// writeMarshal writes a struct's or newtype's MarshalJSON, when it holds a
// collection to fill.
func (g *generator) writeMarshal(b *strings.Builder, decl *ir.Decl, name string) error {
	if !g.marshals(decl) {
		return nil
	}
	if err := g.marshalProblem(decl); err != nil {
		return err
	}
	if err := g.marshalCollides(decl, name, decl.Fields()); err != nil {
		return err
	}
	var body strings.Builder
	recv, self := receiver(decl, name), name+typeArgs(decl)
	if nt := decl.GetNewtype(); nt != nil {
		base, err := g.goType(nt.GetBase())
		if err != nil {
			return err
		}
		fmt.Fprintf(&body, "self := %s(%s)\n", base, recv)
		if err := g.fill(&body, "self", nt.GetBase(), nil, 0); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(&body, "type plain %s\nself := plain(%s)\n", self, recv)
		if _, err := g.fillFields(&body, decl.Fields()); err != nil {
			return err
		}
	}
	g.use("encoding/json")
	fmt.Fprintf(b, "\n// MarshalJSON encodes a %s%s\n", name, fillDoc)
	fmt.Fprintf(b, "func (%s %s) MarshalJSON() ([]byte, error) {\n%sreturn json.Marshal(self)\n}\n", recv, self, body.String())
	return nil
}

// warnParamFills warns when a generic declaration is applied to a type
// holding a collection: its MarshalJSON writes a parameter as encoding/json
// does, a nil slice or map as null.
func (g *generator) warnParamFills(id *ir.ID, fr *frame, at *ir.Position) bool {
	t, decl, fr, ok := g.resolveFill(id, fr)
	if !ok {
		return false
	}
	for _, a := range t.GetArgs() {
		if emit.IsOwn(decl) && g.needsFill(a, fr) {
			g.Warn(emit.Unsupported(at,
				"%s is applied to a type holding a list or map, and its JSON methods write its parameters as plain values, so an empty one there is null",
				decl.GetMeta().GetName()))
			return true
		}
		if g.warnParamFills(a, fr, at) {
			return true
		}
	}
	return false
}
