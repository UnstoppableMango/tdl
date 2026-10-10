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
// gets an UnmarshalJSON that decodes those fields with it.

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
		fmt.Fprintf(b, "\n// MarshalJSON encodes a %s with its %s.\n", variant, strconv.Quote(disc))
		fmt.Fprintf(b, "func (%s %s) MarshalJSON() ([]byte, error) {\n", recv, variant)
		fmt.Fprintf(b, "type plain %s\nreturn json.Marshal(struct {\nTDLDiscriminant string `json:%s`\nplain\n}{%s, plain(%s)})\n}\n",
			variant, strconv.Quote(disc), strconv.Quote(emit.Tag(v)), recv)
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
