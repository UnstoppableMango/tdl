package ast

import (
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

	for _, decl := range file.Decls {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		for _, line := range Doc(decl) {
			b.WriteString(strings.TrimRight("/// "+line, " ") + "\n")
		}
		b.WriteString(printDecl(decl))
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
	default:
		return ""
	}
}

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
		if len(t.Args) > 0 {
			args := make([]string, len(t.Args))
			for i, a := range t.Args {
				args[i] = printTypeRef(a)
			}
			s += "<" + strings.Join(args, ", ") + ">"
		}
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
