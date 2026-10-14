package csharp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// key renders an entity's identity, which a `key` directive names: a Key
// method returning the one property it names, or one building a key record
// struct holding each, which after declares beside the entity. On error
// the caller warns and emits the entity without its key.
func (g *generator) key(d *ir.Decl, typ string, props []property, dir *ir.Directive) (method, after string, err error) {
	pos := dir.GetPosition()
	at := func(arg *ir.Literal) *ir.Position {
		if p := arg.GetPosition(); p != nil {
			return p
		}
		return pos
	}
	name := d.GetMeta().GetName()
	if d.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_ENTITY {
		return "", "", emit.Lost(emit.LossKey, pos, "%s has a key, and only an entity is identified by one", name)
	}
	for _, p := range props {
		if p.name == "Key" {
			return "", "", emit.Lost(emit.LossKey, pos, "%s has a property Key, which the generated key method would be named", name)
		}
	}

	var keyed []property
	for _, arg := range dir.GetArgs() {
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
			return "", "", emit.Lost(emit.LossKey, at(arg), "a key names fields, and %q is %s", arg.GetText(), ir.KindName(arg.GetKind()))
		}
		i := slices.IndexFunc(props, func(p property) bool { return p.field.GetMeta().GetName() == arg.GetText() })
		if i < 0 {
			return "", "", emit.Lost(emit.LossKey, at(arg), "%s has no field %s to key on", name, arg.GetText())
		}
		p := props[i]
		if slices.ContainsFunc(keyed, func(k property) bool { return k.field == p.field }) {
			return "", "", emit.Lost(emit.LossKey, at(arg), "the key of %s names %s twice", name, arg.GetText())
		}
		if p.typ.cmp != "" {
			return "", "", emit.Lost(emit.LossKey, at(arg),
				"%s.%s cannot be part of a key in C#, which compares a collection or bytes by reference", name, arg.GetText())
		}
		keyed = append(keyed, p)
	}
	if len(keyed) == 0 {
		return "", "", emit.Lost(emit.LossKey, pos, "the key of %s names no field", name)
	}

	if len(keyed) == 1 {
		return fmt.Sprintf("public %s Key() => %s;", keyed[0].typ.text, keyed[0].name), "", nil
	}

	record := strings.TrimPrefix(typ, "@") + "Key"
	if other, ok := g.owners[record]; ok {
		return "", "", emit.Lost(emit.LossKey, pos, "the key record %s would collide with %s", record, other)
	}
	params := make([]string, len(keyed))
	args := make([]string, len(keyed))
	for i, p := range keyed {
		if strings.TrimPrefix(p.name, "@") == record {
			return "", "", emit.Lost(emit.LossKey, pos, "the key record %s cannot hold a property of its own name", record)
		}
		params[i] = p.typ.text + " " + p.name
		args[i] = p.name
	}
	after = fmt.Sprintf("\n/// <summary>The fields identifying a <see cref=\"%s\"/>.</summary>\npublic readonly partial record struct %s(%s);\n",
		typ, record, strings.Join(params, ", "))
	method = fmt.Sprintf("public %s Key() => new(%s);", record, strings.Join(args, ", "))
	return method, after, nil
}
