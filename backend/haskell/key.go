package haskell

import (
	"fmt"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// key renders an entity's identity, which a `key` directive names: a
// function returning the one field it names, or a key record holding every
// field and a function building it. It returns the type and constructor a
// key record declares, and its labels. On error the caller warns and emits
// the entity without its key.
func (g *generator) key(b *strings.Builder, d *ir.Decl, typ string, dir *ir.Directive) (string, []string, error) {
	pos := dir.GetPosition()
	at := func(arg *ir.Literal) *ir.Position {
		if p := arg.GetPosition(); p != nil {
			return p
		}
		return pos
	}
	name := d.GetMeta().GetName()
	if d.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_ENTITY {
		return "", nil, emit.Lost(emit.LossKey, pos, "%s has a key, and only an entity is identified by one", name)
	}

	type keyField struct{ label, typ string }
	var fields []keyField
	var seen []*ir.Field
	for _, arg := range dir.GetArgs() {
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
			return "", nil, emit.Lost(emit.LossKey, at(arg), "a key names fields, and %q is %s", arg.GetText(), ir.KindName(arg.GetKind()))
		}
		i := slices.IndexFunc(d.Fields(), func(f *ir.Field) bool { return f.GetMeta().GetName() == arg.GetText() })
		if i < 0 {
			return "", nil, emit.Lost(emit.LossKey, at(arg), "%s has no field %s to key on", name, arg.GetText())
		}
		f := d.Fields()[i]
		if slices.Contains(seen, f) {
			return "", nil, emit.Lost(emit.LossKey, at(arg), "the key of %s names %s twice", name, arg.GetText())
		}
		seen = append(seen, f)
		label, err := g.fieldName(name, f)
		if err != nil {
			return "", nil, err
		}
		t, err := g.typ(f.GetType())
		if err != nil {
			return "", nil, err
		}
		fields = append(fields, keyField{label, t.text})
	}
	if len(fields) == 0 {
		return "", nil, emit.Lost(emit.LossKey, pos, "the key of %s names no field", name)
	}

	fn := lowerFirst(typ) + "Key"
	v := strings.ToLower(typ[:1])
	if len(fields) == 1 {
		fmt.Fprintf(b, "\n-- | What identifies '%s' values.\n", typ)
		fmt.Fprintf(b, "%s :: %s%s -> %s\n", fn, g.requiresContext(d), applyTo(typ, d).text, fields[0].typ)
		fmt.Fprintf(b, "%s %s = %s.%s\n", fn, v, v, fields[0].label)
		return "", nil, nil
	}

	if len(d.Params()) > 0 {
		return "", nil, emit.Lost(emit.LossKey, pos, "%s is parameterized, and a key record of several fields is not generated for one", name)
	}
	record := typ + "Key"
	if other, ok := g.names[record]; ok {
		return "", nil, emit.Lost(emit.LossKey, pos, "the key record %s would collide with %s", record, other)
	}
	decls := make([]string, len(fields))
	inits := make([]string, len(fields))
	for i, f := range fields {
		decls[i] = f.label + " :: " + f.typ
		inits[i] = f.label + " = " + v + "." + f.label
	}
	fmt.Fprintf(b, "\n-- | The fields identifying '%s' values.\n", typ)
	fmt.Fprintf(b, "data %s = %s {%s}\n", record, record, strings.Join(decls, ", "))
	b.WriteString(g.deriving(eq, ord, show, generic))
	fmt.Fprintf(b, "\n-- | What identifies '%s' values.\n", typ)
	fmt.Fprintf(b, "%s :: %s -> %s\n", fn, typ, record)
	fmt.Fprintf(b, "%s %s = %s {%s}\n", fn, v, record, strings.Join(inits, ", "))
	labels := make([]string, len(fields))
	for i, f := range fields {
		labels[i] = f.label
	}
	return record, labels, nil
}
