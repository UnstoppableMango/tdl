package ast

import (
	"strconv"
	"strings"
)

// Fprint renders file as canonical TDL source, the formatting produced by
// `tdl fmt`. It is idempotent: formatting canonical output changes nothing.
//
// Whitespace is insignificant in TDL, so the formatter owns layout
// entirely. Its decisions depend only on the tree, never on how the input
// was written, which is what makes idempotence hold.
func Fprint(file *File) string {
	var b strings.Builder

	if file.Package != nil {
		b.WriteString("package " + file.Package.Path + "\n")
	}

	if len(file.Imports) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		for _, imp := range file.Imports {
			for _, line := range imp.Doc {
				b.WriteString(strings.TrimRight("/// "+line, " ") + "\n")
			}
			b.WriteString("import " + quote(imp.Path) + " as " + imp.Alias + "\n")
		}
	}

	prevMultiline := true // force a blank line after the header, if any
	for _, decl := range file.Decls {
		text := printDecl(decl)
		multiline := strings.Count(text, "\n") > 1
		annotated := len(Doc(decl)) > 0 || Deprecated(decl) != nil

		// Consecutive one-line declarations group together; anything with a
		// body, a doc comment, or a deprecation gets a blank line of its own.
		// The decision reads only the tree, so formatting stays idempotent.
		if b.Len() > 0 && (multiline || prevMultiline || annotated) {
			b.WriteString("\n")
		}

		writeDoc(&b, "", Doc(decl))
		if dep := Deprecated(decl); dep != nil {
			b.WriteString(printDeprecated(dep) + "\n")
		}
		b.WriteString(text)
		prevMultiline = multiline || annotated
	}

	return b.String()
}

func printDecl(decl Decl) string {
	switch d := decl.(type) {
	case *PrimitiveDecl:
		if d.Kind != nil {
			return "primitive " + d.N + ": " + printKind(d.Kind) + "\n"
		}
		return "primitive " + d.N + "\n"

	case *AliasDecl:
		return "alias " + d.N + printParams(d.Params) + " = " + printTypeRef(d.Target) + "\n"

	case *NewtypeDecl:
		return "type " + d.N + printParams(d.Params) + ": " + printTypeRef(d.Base) +
			printRequires(d.Requires) + printConstraints(d.Constraints, "") + "\n"

	case *StructDecl:
		head := d.Keyword + " " + d.N + printParams(d.Params) +
			printConforms(d.Conforms) + printRequires(d.Requires)
		return head + printMembers(d.Members)

	case *EnumDecl:
		head := "enum " + d.N + printParams(d.Params) +
			printConforms(d.Conforms) + printRequires(d.Requires)
		return head + printVariants(d.Variants)

	case *ClassDecl:
		head := "class " + d.N + printParams(d.Params) + printFunDeps(d.FunDeps) +
			printConforms(d.Conforms) + printRequires(d.Requires)
		return head + printMembers(d.Members)

	case *InstanceDecl:
		return printInstance(d)

	case *UnitDecl:
		if d.Expr != nil {
			return "unit " + d.N + " = " + printUnitExpr(d.Expr) + "\n"
		}
		return "unit " + d.N + "\n"

	case *TargetDecl:
		return "target " + d.N + " for " + d.For + printEntries(d.Entries, "")

	default:
		return ""
	}
}

func writeDoc(b *strings.Builder, indent string, doc []string) {
	for _, line := range doc {
		b.WriteString(strings.TrimRight(indent+"/// "+line, " ") + "\n")
	}
}

func printDeprecated(dep *Deprecation) string {
	if dep.Reason != "" {
		return "deprecated(" + quote(dep.Reason) + ")"
	}
	return "deprecated"
}

func printConforms(refs []*ClassRef) string {
	if len(refs) == 0 {
		return ""
	}
	return ": " + printClassRefs(refs)
}

func printRequires(refs []*ClassRef) string {
	if len(refs) == 0 {
		return ""
	}
	return " requires " + printClassRefs(refs)
}

func printClassRefs(refs []*ClassRef) string {
	parts := make([]string, len(refs))
	for i, r := range refs {
		parts[i] = r.N
		if r.Qualifier != "" {
			parts[i] = r.Qualifier + "." + parts[i]
		}
		parts[i] += printTypeArgs(r.Args)
	}
	return strings.Join(parts, ", ")
}

func printFunDeps(deps []*FunDep) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = strings.Join(d.From, " ") + " -> " + strings.Join(d.To, " ")
	}
	return " | " + strings.Join(parts, ", ")
}

// printInstance keeps the form that was written. `instance C for T` is
// sugar for `instance C<T>`, and rewriting one into the other would lose
// what the author chose to say.
func printInstance(d *InstanceDecl) string {
	s := "instance "
	if len(d.Params) > 0 {
		s += printParams(d.Params) + " "
	}
	s += printClassRefs([]*ClassRef{d.Class})
	if d.For != nil {
		s += " for " + printTypeRef(d.For)
	}
	s += printRequires(d.Requires)

	if len(d.Binds) == 0 {
		return s + "\n"
	}

	var b strings.Builder
	b.WriteString(s + " {\n")
	for _, bind := range d.Binds {
		b.WriteString("  type " + bind.N + " = " + printTypeRef(bind.Target) + "\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func printMembers(members []Member) string {
	if len(members) == 0 {
		return " { }\n"
	}

	var b strings.Builder
	b.WriteString(" {\n")
	for _, m := range members {
		switch n := m.(type) {
		case *Include:
			b.WriteString("  include " + printClassRefs([]*ClassRef{n.Type}) + "\n")
		case *KeyRequirement:
			b.WriteString("  key\n")
		case *AssocTypeReq:
			writeDoc(&b, "  ", n.Doc)
			b.WriteString("  type " + n.N)
			if n.Kind != nil {
				b.WriteString(": " + printKind(n.Kind))
			}
			b.WriteString("\n")
		case *Field:
			writeDoc(&b, "  ", n.Doc)
			b.WriteString("  " + printField(n) + "\n")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func printField(f *Field) string {
	s := printFieldHead(f) + printConstraints(f.Constraints, "  ")
	if f.Default != nil {
		s += " = " + printLiteral(f.Default)
	}
	return s
}

// printFieldHead renders everything up to the constraint block. [Dump] uses
// it so a field stays one line even when its constraints do not.
func printFieldHead(f *Field) string {
	var s string
	if f.Key {
		s += "key "
	}
	if f.Dep != nil {
		s += printDeprecated(f.Dep) + " "
	}
	s += f.N + ": " + printTypeRef(f.Type)
	if f.Owned {
		s += " owned"
	}
	return s
}

func printVariants(variants []*Variant) string {
	if len(variants) == 0 {
		return " { }\n"
	}

	// A block stays on one line when it fits the column limit. Whitespace is
	// insignificant, so this depends only on content: the same tree always
	// formats the same way, whatever the input looked like.
	inline := true
	oneLine := " {"
	for _, v := range variants {
		if len(v.Fields) > 0 || len(v.Doc) > 0 || v.Dep != nil {
			inline = false
			break
		}
		oneLine += " " + v.N
	}
	if inline && len(oneLine)+2 <= columnLimit {
		return oneLine + " }\n"
	}

	var b strings.Builder
	b.WriteString(" {\n")
	for _, v := range variants {
		writeDoc(&b, "  ", v.Doc)
		b.WriteString("  ")
		if v.Dep != nil {
			b.WriteString(printDeprecated(v.Dep) + " ")
		}
		b.WriteString(v.N)
		if len(v.Fields) > 0 {
			parts := make([]string, len(v.Fields))
			for i, f := range v.Fields {
				parts[i] = printField(f)
			}
			b.WriteString(" { " + strings.Join(parts, " ") + " }")
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func printEntries(entries []*TargetEntry, indent string) string {
	if len(entries) == 0 {
		return " { }\n"
	}

	var b strings.Builder
	b.WriteString(" {\n")
	inner := indent + "  "
	for _, e := range entries {
		b.WriteString(inner)
		switch {
		case e.Entries != nil:
			b.WriteString(e.Path)
			b.WriteString(strings.TrimSuffix(printEntries(e.Entries, inner), "\n"))
			b.WriteString("\n")
		case e.Path != "":
			b.WriteString(e.Path + " => " + printDirective(e.Directive) + "\n")
		default:
			b.WriteString(printDirective(e.Directive) + "\n")
		}
	}
	b.WriteString(indent + "}\n")
	return b.String()
}

func printDirective(d *Directive) string {
	if len(d.Args) == 0 {
		return d.N
	}
	args := make([]string, len(d.Args))
	for i, a := range d.Args {
		args[i] = printLiteral(a)
	}
	return d.N + "(" + strings.Join(args, ", ") + ")"
}

func printLiteral(l *Literal) string {
	switch l.Kind {
	case LitString:
		return quote(l.Text)
	case LitList:
		items := make([]string, len(l.Items))
		for i, it := range l.Items {
			items[i] = printLiteral(it)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case LitRegex:
		return "/" + l.Text + "/"
	case LitRange:
		var lo, hi string
		if l.Lo != nil {
			lo = l.Lo.Text
		}
		if l.Hi != nil {
			hi = l.Hi.Text
		}
		return lo + ".." + hi
	default:
		return l.Text
	}
}

// printConstraints renders a `where { ... }` block.
//
// A single constraint stays on one line when it fits the column limit; two
// or more always expand. One constraint is an aside on the thing it
// qualifies, several are a specification of it, and the layout says which.
func printConstraints(cs []*Constraint, indent string) string {
	if len(cs) == 0 {
		return ""
	}

	if len(cs) == 1 {
		if line := " where { " + printConstraint(cs[0]) + " }"; len(indent)+len(line) <= columnLimit {
			return line
		}
	}

	var b strings.Builder
	b.WriteString(" where {\n")
	for _, c := range cs {
		b.WriteString(indent + "  " + printConstraint(c) + "\n")
	}
	b.WriteString(indent + "}")
	return b.String()
}

func printConstraint(c *Constraint) string {
	if len(c.Args) == 0 {
		return c.N
	}
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = printLiteral(a)
	}
	return c.N + "(" + strings.Join(args, ", ") + ")"
}

// columnLimit is the width a block must fit within to stay on one line.
const columnLimit = 80

func printParams(params []*TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.N
		if p.Kind != nil {
			parts[i] += ": " + printKind(p.Kind)
		}
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

func printKind(k *Kind) string {
	var s string
	if k.Paren != nil {
		s = "(" + printKind(k.Paren) + ")"
	} else {
		s = k.N
	}
	if k.Arrow != nil {
		s += " -> " + printKind(k.Arrow)
	}
	return s
}

func printTypeArgs(args []*TypeArg) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, len(args))
	for i, a := range args {
		if a.Unit != nil {
			parts[i] = printUnitExpr(a.Unit)
		} else {
			parts[i] = printTypeRef(a.Type)
		}
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// PrintUnitExpr renders a unit expression the way `tdl fmt` writes it.
//
// Exported because lowering records what a unit was written as beside what
// it reduces to, and reconstructing the text there would be a second
// printer to keep in step with this one.
func PrintUnitExpr(e *UnitExpr) string {
	return printUnitExpr(e)
}

// printUnitExpr renders a unit expression without spaces around its
// operators, the form the spec uses: `kg*m/s^2`.
func printUnitExpr(e *UnitExpr) string {
	var b strings.Builder
	for _, t := range e.Terms {
		b.WriteString(t.Op)
		if t.Paren != nil {
			b.WriteString("(" + printUnitExpr(t.Paren) + ")")
		} else {
			b.WriteString(t.N)
		}
		if t.Exp != 1 {
			b.WriteString("^" + strconv.Itoa(t.Exp))
		}
	}
	return b.String()
}

func printTypeRef(t *TypeRef) string {
	if t == nil {
		return ""
	}

	var s string
	switch {
	case t.List != nil:
		s = "[" + printTypeRef(t.List) + "]"
	case t.Set != nil:
		s = "{" + printTypeRef(t.Set) + "}"
	case t.MapKey != nil:
		s = "{" + printTypeRef(t.MapKey) + " -> " + printTypeRef(t.MapValue) + "}"
	default:
		s = t.N
		if t.Qualifier != "" {
			s = t.Qualifier + "." + s
		}
		s += printTypeArgs(t.Args)
	}

	if t.Optional {
		s += "?"
	}
	if t.Nullable {
		s += " | null"
	}
	return s
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
