package ast

import (
	"fmt"
	"strings"
)

// Dump renders file as an indented tree, one node per line, with the
// source position of each node. Where [Fprint] answers "what does this
// look like as TDL", Dump answers "what did the parser actually build".
// It is a debugging and exploration aid; the exact layout is not stable
// and nothing parses it back.
func Dump(file *File) string {
	var b strings.Builder

	name := file.Filename
	if name == "" {
		name = "<input>"
	}
	fmt.Fprintf(&b, "File %s\n", name)

	var nodes []func(prefix string, last bool)

	if file.Package != nil {
		pkg := file.Package
		nodes = append(nodes, func(prefix string, last bool) {
			leaf(&b, prefix, last, "Package "+pkg.Name, pkg.Pos)
		})
	}
	for _, imp := range file.Imports {
		nodes = append(nodes, func(prefix string, last bool) {
			leaf(&b, prefix, last, fmt.Sprintf("Import %q as %s", imp.Path, imp.Alias), imp.Pos)
		})
	}
	for _, ty := range file.Types {
		nodes = append(nodes, func(prefix string, last bool) {
			dumpType(&b, prefix, last, ty)
		})
	}
	for _, en := range file.Enums {
		nodes = append(nodes, func(prefix string, last bool) {
			dumpEnum(&b, prefix, last, en)
		})
	}

	for i, node := range nodes {
		node("", i == len(nodes)-1)
	}
	return b.String()
}

func dumpType(b *strings.Builder, prefix string, last bool, ty *TypeDecl) {
	leaf(b, prefix, last, "Type "+ty.Name, ty.Pos)
	inner := prefix + branchPad(last)

	total := len(ty.Annotations) + len(ty.Fields)
	i := 0
	for _, ann := range ty.Annotations {
		dumpAnnotation(b, inner, i == total-1, ann)
		i++
	}
	for _, f := range ty.Fields {
		dumpField(b, inner, i == total-1, f)
		i++
	}
}

func dumpField(b *strings.Builder, prefix string, last bool, f *Field) {
	desc := fmt.Sprintf("Field %s: %s", f.Name, printTypeRef(f.Type))
	if f.Optional {
		desc += "?"
	}
	if f.Default != nil {
		desc += " = " + printLiteral(f.Default)
	}
	leaf(b, prefix, last, desc, f.Pos)

	inner := prefix + branchPad(last)
	for i, ann := range f.Annotations {
		dumpAnnotation(b, inner, i == len(f.Annotations)-1, ann)
	}
}

func dumpEnum(b *strings.Builder, prefix string, last bool, en *EnumDecl) {
	leaf(b, prefix, last, "Enum "+en.Name, en.Pos)
	inner := prefix + branchPad(last)

	total := len(en.Annotations) + len(en.Values)
	i := 0
	for _, ann := range en.Annotations {
		dumpAnnotation(b, inner, i == total-1, ann)
		i++
	}
	for _, v := range en.Values {
		desc := "Value " + v.Name
		if v.Value != nil {
			desc += " = " + printLiteral(v.Value)
		}
		leaf(b, inner, i == total-1, desc, v.Pos)
		valueInner := inner + branchPad(i == total-1)
		for j, ann := range v.Annotations {
			dumpAnnotation(b, valueInner, j == len(v.Annotations)-1, ann)
		}
		i++
	}
}

func dumpAnnotation(b *strings.Builder, prefix string, last bool, ann *Annotation) {
	var args strings.Builder
	for i, arg := range ann.Args {
		if i > 0 {
			args.WriteString(", ")
		}
		args.WriteString(arg.Name)
		args.WriteString(": ")
		args.WriteString(printLiteral(arg.Value))
	}
	desc := "@" + ann.Namespace
	if len(ann.Args) > 0 {
		desc += "(" + args.String() + ")"
	}
	leaf(b, prefix, last, desc, ann.Pos)
}

// leaf writes one tree line: branch mark, description, source position.
func leaf(b *strings.Builder, prefix string, last bool, desc string, pos Position) {
	fmt.Fprintf(b, "%s%s%s  %d:%d\n", prefix, branchMark(last), desc, pos.Line, pos.Col)
}

func branchMark(last bool) string {
	if last {
		return "`- "
	}
	return "|- "
}

func branchPad(last bool) string {
	if last {
		return "   "
	}
	return "|  "
}
