package ir

import (
	"fmt"
	"strconv"
	"strings"
)

// Dump renders a model as an indented tree, one node per line: the output
// of `tdl ir`.
//
// It matches the conventions of `tdl ast`, so the two can be read side by
// side to see what resolution did.
func Dump(m *Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Model %s\n", m.GetPackage())

	d := &dumper{model: m, b: &b}
	if len(m.GetImports()) > 0 {
		d.section("Imports", true, func(prefix string) {
			for i, imp := range m.GetImports() {
				desc := imp.GetPath() + " as " + imp.GetAlias()
				if pkg := imp.GetPackage(); pkg != "" {
					desc += "  (" + pkg + ")"
				}
				d.leaf(prefix, i == len(m.GetImports())-1, desc, imp.GetPosition())
			}
		})
	}
	d.section("Decls", len(m.GetTypes()) > 0 || len(m.GetExterns()) > 0, func(prefix string) {
		for i, decl := range m.GetDecls() {
			d.decl(prefix, i == len(m.GetDecls())-1, i, decl)
		}
	})
	if len(m.GetInstances()) > 0 {
		d.section("Instances", true, func(prefix string) {
			for i, inst := range m.GetInstances() {
				last := i == len(m.GetInstances())-1
				d.leaf(prefix, last, d.instanceLine(inst), inst.GetMeta().GetPosition())
				for j, b := range inst.GetBinds() {
					d.leaf(prefix+pad(last), j == len(inst.GetBinds())-1,
						"type "+b.GetName()+" = "+d.ref(b.GetType()), b.GetPosition())
				}
			}
		})
	}
	if len(m.GetSatisfies()) > 0 {
		d.section("Satisfies", true, func(prefix string) {
			for i, sat := range m.GetSatisfies() {
				names := make([]string, 0, len(sat.GetDecls())+len(sat.GetTypes()))
				for _, id := range sat.GetDecls() {
					names = append(names, id.GetName())
				}
				for _, id := range sat.GetTypes() {
					names = append(names, id.GetName())
				}
				line := sat.GetClass().GetName() + ": nothing"
				if len(names) > 0 {
					line = sat.GetClass().GetName() + ": " + strings.Join(names, ", ")
				}
				d.leaf(prefix, i == len(m.GetSatisfies())-1, line, nil)
			}
		})
	}
	d.section("Types", len(m.GetExterns()) > 0, func(prefix string) {
		for i, t := range m.GetTypes() {
			d.leaf(prefix, i == len(m.GetTypes())-1, d.typeLine(i, t), t.GetPosition())
		}
	})
	if len(m.GetExterns()) > 0 {
		d.section("Externs", false, func(prefix string) {
			for i, e := range m.GetExterns() {
				d.leaf(prefix, i == len(m.GetExterns())-1, e.GetPackage()+"."+e.GetName(), e.GetPosition())
			}
		})
	}
	return b.String()
}

type dumper struct {
	model *Model
	b     *strings.Builder
}

func (d *dumper) section(title string, more bool, body func(prefix string)) {
	d.b.WriteString(branch(!more) + title + "\n")
	body(pad(!more))
}

func (d *dumper) decl(prefix string, last bool, index int, decl *Decl) {
	d.leaf(prefix, last, fmt.Sprintf("[%d] %s", index, declLine(decl)), decl.GetMeta().GetPosition())

	inner := prefix + pad(last)
	var kids []func(bool)

	for _, p := range decl.Params() {
		kids = append(kids, func(l bool) {
			d.leaf(inner, l, "param "+p.GetName()+kindSuffix(p.GetKind()), p.GetPosition())
		})
	}
	switch {
	case decl.GetAlias() != nil:
		id := decl.GetAlias().GetTarget()
		kids = append(kids, func(l bool) { d.leaf(inner, l, "target "+d.ref(id), nil) })
	case decl.GetNewtype() != nil:
		n := decl.GetNewtype()
		kids = append(kids, func(l bool) { d.leaf(inner, l, "base "+d.ref(n.GetBase()), nil) })
		for _, c := range n.GetValueConstraints() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, constraintText(c), c.GetPosition()) })
		}
	case decl.GetStructure() != nil:
		for _, r := range decl.GetStructure().GetConforms() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, "conforms "+d.classRef(r), r.GetPosition()) })
		}
		for _, f := range decl.GetStructure().GetFields() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, d.fieldLine(f), f.GetMeta().GetPosition()) })
		}
	case decl.GetClass() != nil:
		c := decl.GetClass()
		for _, r := range c.GetRequiresClasses() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, "requires "+d.classRef(r), r.GetPosition()) })
		}
		for _, fd := range c.GetFunDeps() {
			kids = append(kids, func(l bool) {
				d.leaf(inner, l, "fundep "+strings.Join(fd.GetFrom(), " ")+" -> "+strings.Join(fd.GetTo(), " "), fd.GetPosition())
			})
		}
		if c.GetRequiresKey() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, "requires key", nil) })
		}
		for _, at := range c.GetAssocTypes() {
			kids = append(kids, func(l bool) {
				d.leaf(inner, l, "type "+at.GetMeta().GetName()+kindSuffix(at.GetKind()), at.GetMeta().GetPosition())
			})
		}
		for _, f := range c.GetFields() {
			kids = append(kids, func(l bool) { d.leaf(inner, l, d.fieldLine(f), f.GetMeta().GetPosition()) })
		}

	case decl.GetEnumeration() != nil:
		for _, v := range decl.GetEnumeration().GetVariants() {
			kids = append(kids, func(l bool) {
				d.leaf(inner, l, "variant "+v.GetMeta().GetName()+deprecatedSuffix(v.GetMeta()), v.GetMeta().GetPosition())
				for i, f := range v.GetFields() {
					d.leaf(inner+pad(l), i == len(v.GetFields())-1, d.fieldLine(f), f.GetMeta().GetPosition())
				}
			})
		}
	}

	for i, kid := range kids {
		kid(i == len(kids)-1)
	}
}

func declLine(decl *Decl) string {
	name := decl.GetMeta().GetName()
	var kind string
	switch {
	case decl.GetPrimitive() != nil:
		return "primitive " + name + kindSuffix(decl.GetPrimitive().GetKind()) + deprecatedSuffix(decl.GetMeta())
	case decl.GetAlias() != nil:
		kind = "alias"
	case decl.GetNewtype() != nil:
		kind = "newtype"
	case decl.GetEnumeration() != nil:
		kind = "enum"
	case decl.GetClass() != nil:
		kind = "class"
	case decl.GetClass() != nil:
		kind = "class"
	case decl.GetStructure() != nil:
		switch decl.GetStructure().GetKind() {
		case StructKind_STRUCT_KIND_ENTITY:
			kind = "entity"
		case StructKind_STRUCT_KIND_VALUE:
			kind = "value"
		case StructKind_STRUCT_KIND_MIXIN:
			kind = "mixin"
		default:
			kind = "struct"
		}
	default:
		kind = "unlowered"
	}
	return kind + " " + name + deprecatedSuffix(decl.GetMeta())
}

func (d *dumper) fieldLine(f *Field) string {
	var mods string
	if f.GetKey() {
		mods += "key "
	}
	if f.GetOwned() {
		mods += "owned "
	}
	line := "field " + mods + f.GetMeta().GetName() + ": " + d.ref(f.GetType())
	for _, c := range f.GetConstraints() {
		line += "  " + constraintText(c)
	}
	if def := f.GetDefaultValue(); def != nil {
		line += " = " + literalText(def)
	}
	if from := f.GetIncludedFrom(); from != nil {
		line += "  (from " + from.GetName() + ")"
	}
	return line + deprecatedSuffix(f.GetMeta())
}

func (d *dumper) instanceLine(inst *Instance) string {
	line := "instance " + d.classRef(inst.GetClass())
	if len(inst.GetParams()) > 0 {
		names := make([]string, len(inst.GetParams()))
		for i, p := range inst.GetParams() {
			names[i] = p.GetName()
		}
		line = "instance <" + strings.Join(names, ", ") + "> " + d.classRef(inst.GetClass())
	}
	if len(inst.GetRequires()) > 0 {
		reqs := make([]string, len(inst.GetRequires()))
		for i, r := range inst.GetRequires() {
			reqs[i] = d.classRef(r)
		}
		line += " requires " + strings.Join(reqs, ", ")
	}
	return line
}

func (d *dumper) classRef(r *ClassRef) string {
	name := r.GetClass().GetName()
	if e := r.GetExtern(); e != nil {
		name = e.GetName()
	}
	if len(r.GetArgs()) == 0 {
		return name
	}
	args := make([]string, len(r.GetArgs()))
	for i, a := range r.GetArgs() {
		if !a.Resolved() {
			args[i] = "?"
			continue
		}
		args[i] = d.render(d.model.Type(a))
	}
	return name + "<" + strings.Join(args, ", ") + ">"
}

func (d *dumper) typeLine(index int, t *Type) string {
	origin := "-> decls[" + itoa(int(t.GetCtor().GetIndex())) + "]"
	switch {
	case t.GetParam() != nil:
		origin = "-> param " + t.GetParam().GetOwner().GetName() + "." + t.GetParam().GetName()
	case t.GetExtern() != nil:
		origin = "-> externs[" + itoa(int(t.GetExtern().GetIndex())) + "]"
	case !t.GetCtor().Resolved():
		origin = "-> unresolved"
	}
	return fmt.Sprintf("[%d] %s  %s  %s", index, d.render(t), form(t.GetWrote()), origin)
}

// ref renders a type by index and by what it says, so a field line reads
// without chasing the table.
func (d *dumper) ref(id *ID) string {
	if !id.Resolved() {
		return "?"
	}
	return fmt.Sprintf("types[%d] %s", id.GetIndex(), d.render(d.model.Type(id)))
}

// render expands a type through the table. Arguments are always interned
// before the type that holds them, so the walk terminates.
func (d *dumper) render(t *Type) string {
	if t == nil {
		return "?"
	}

	name := t.GetCtor().GetName()
	if p := t.GetParam(); p != nil {
		name = p.GetName()
	}
	if e := t.GetExtern(); e != nil {
		name = e.GetName()
	}
	if name == "" {
		name = "?"
	}
	if len(t.GetArgs()) == 0 {
		return name
	}

	args := make([]string, len(t.GetArgs()))
	for i, a := range t.GetArgs() {
		if !a.Resolved() {
			args[i] = "?"
			continue
		}
		args[i] = d.render(d.model.Type(a))
	}
	return name + "<" + strings.Join(args, ", ") + ">"
}

// constraintText renders a constraint, noting the newtype it came from
// when it was inherited rather than written here.
func constraintText(c *Constraint) string {
	text := c.GetName()
	if len(c.GetArgs()) > 0 {
		args := make([]string, len(c.GetArgs()))
		for i, a := range c.GetArgs() {
			args[i] = literalText(a)
		}
		text += "(" + strings.Join(args, ", ") + ")"
	}
	if from := c.GetFrom(); from != nil {
		text += " (from " + from.GetName() + ")"
	}
	return text
}

func literalText(l *Literal) string {
	switch l.GetKind() {
	case LiteralKind_LITERAL_KIND_STRING:
		return strconv.Quote(l.GetText())
	case LiteralKind_LITERAL_KIND_REGEX:
		return "/" + l.GetText() + "/"
	case LiteralKind_LITERAL_KIND_LIST:
		items := make([]string, len(l.GetItems()))
		for i, item := range l.GetItems() {
			items[i] = literalText(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case LiteralKind_LITERAL_KIND_RANGE:
		var lo, hi string
		if l.GetRange().Low != nil {
			lo = strconv.FormatInt(l.GetRange().GetLow(), 10)
		}
		if l.GetRange().High != nil {
			hi = strconv.FormatInt(l.GetRange().GetHigh(), 10)
		}
		return lo + ".." + hi
	case LiteralKind_LITERAL_KIND_NAME:
		if v := l.GetVariant(); v != nil {
			return l.GetText()
		}
		return l.GetText() + " (unresolved)"
	}
	return l.GetText()
}

func form(f SyntacticForm) string {
	switch f {
	case SyntacticForm_SYNTACTIC_FORM_NAMED:
		return "named"
	case SyntacticForm_SYNTACTIC_FORM_BRACKETS:
		return "[T]"
	case SyntacticForm_SYNTACTIC_FORM_BRACES:
		return "{T}"
	case SyntacticForm_SYNTACTIC_FORM_ARROW:
		return "{K -> V}"
	case SyntacticForm_SYNTACTIC_FORM_QUESTION:
		return "T?"
	case SyntacticForm_SYNTACTIC_FORM_OR_NULL:
		return "T | null"
	}
	return "unspecified"
}

func kindSuffix(k *Kind) string {
	if k == nil {
		return ""
	}
	return ": " + renderKind(k)
}

func renderKind(k *Kind) string {
	var s string
	if k.GetParen() != nil {
		s = "(" + renderKind(k.GetParen()) + ")"
	} else if k.GetAtom() == KindAtom_KIND_ATOM_UNIT {
		s = "unit"
	} else {
		s = "type"
	}
	if k.GetArrow() != nil {
		s += " -> " + renderKind(k.GetArrow())
	}
	return s
}

func deprecatedSuffix(m *Meta) string {
	if !m.IsDeprecated() {
		return ""
	}
	if r := m.GetDeprecated().GetReason(); r != "" {
		return "  deprecated(" + r + ")"
	}
	return "  deprecated"
}

func (d *dumper) leaf(prefix string, last bool, desc string, pos *Position) {
	d.b.WriteString(prefix + branch(last) + desc)
	if pos != nil {
		fmt.Fprintf(d.b, "  %s:%d:%d", pos.GetFilename(), pos.GetLine(), pos.GetColumn())
	}
	d.b.WriteString("\n")
}

func branch(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

func pad(last bool) string {
	if last {
		return "    "
	}
	return "│   "
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
