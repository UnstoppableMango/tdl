package ast

import (
	"fmt"
	"strconv"
	"strings"
)

// Fprint writes canonical TDL source text for file to a string. This is
// the single source of truth for what `tdl fmt` produces: formatting is
// deterministic and depends only on the parsed tree, never on the
// original source's whitespace.
func Fprint(file *File) string {
	var b strings.Builder

	if file.Package != nil {
		fmt.Fprintf(&b, "package %s\n\n", file.Package.Name)
	}

	if len(file.Imports) > 0 {
		for _, imp := range file.Imports {
			fmt.Fprintf(&b, "import %q as %s\n", imp.Path, imp.Alias)
		}
		b.WriteByte('\n')
	}

	first := true
	sep := func() {
		if !first {
			b.WriteByte('\n')
		}
		first = false
	}

	for _, ty := range file.Types {
		sep()
		printAnnotations(&b, ty.Annotations, "")
		fmt.Fprintf(&b, "type %s {\n", ty.Name)
		for _, f := range ty.Fields {
			printAnnotations(&b, f.Annotations, "  ")
			b.WriteString("  ")
			b.WriteString(f.Name)
			b.WriteString(": ")
			b.WriteString(printTypeRef(f.Type))
			if f.Optional {
				b.WriteByte('?')
			}
			if f.Default != nil {
				b.WriteString(" = ")
				b.WriteString(printLiteral(f.Default))
			}
			b.WriteByte('\n')
		}
		b.WriteString("}\n")
	}

	for _, e := range file.Enums {
		sep()
		printAnnotations(&b, e.Annotations, "")
		fmt.Fprintf(&b, "enum %s {\n", e.Name)
		for _, v := range e.Values {
			printAnnotations(&b, v.Annotations, "  ")
			b.WriteString("  ")
			b.WriteString(v.Name)
			if v.Value != nil {
				b.WriteString(" = ")
				b.WriteString(printLiteral(v.Value))
			}
			b.WriteByte('\n')
		}
		b.WriteString("}\n")
	}

	return b.String()
}

func printAnnotations(b *strings.Builder, anns []*Annotation, indent string) {
	for _, ann := range anns {
		b.WriteString(indent)
		b.WriteByte('@')
		b.WriteString(ann.Namespace)
		if len(ann.Args) > 0 {
			b.WriteByte('(')
			for i, arg := range ann.Args {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(arg.Name)
				b.WriteString(": ")
				b.WriteString(printLiteral(arg.Value))
			}
			b.WriteByte(')')
		}
		b.WriteByte('\n')
	}
}

func printTypeRef(t TypeRef) string {
	switch {
	case t.List != nil:
		return "list<" + printTypeRef(*t.List) + ">"
	case t.MapKey != nil:
		return "map<" + printTypeRef(*t.MapKey) + ", " + printTypeRef(*t.MapValue) + ">"
	case t.Qualifier != "":
		return t.Qualifier + "." + t.Name
	default:
		return t.Name
	}
}

func printLiteral(l *Literal) string {
	if l == nil {
		return ""
	}
	switch l.Kind {
	case LitString:
		return strconv.Quote(l.Str)
	case LitInt:
		return strconv.FormatInt(l.Int, 10)
	case LitFloat:
		return strconv.FormatFloat(l.Float, 'g', -1, 64)
	case LitBool:
		if l.Bool {
			return "true"
		}
		return "false"
	case LitList:
		parts := make([]string, len(l.List))
		for i, e := range l.List {
			parts[i] = printLiteral(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return ""
	}
}
