package ast

import (
	"fmt"
	"strings"
)

// Dump renders file as an indented tree, one node per line, with the source
// position of each. It is the output of `tdl ast`.
func Dump(file *File) string {
	var b strings.Builder
	fmt.Fprintf(&b, "File %s\n", file.Filename)

	type entry struct {
		desc string
		pos  Position
		kids func(prefix string)
	}
	var entries []entry

	if pkg := file.Package; pkg != nil {
		entries = append(entries, entry{fmt.Sprintf("Package %s", pkg.Path), pkg.P, nil})
	}
	for _, imp := range file.Imports {
		entries = append(entries, entry{fmt.Sprintf("Import %q as %s", imp.Path, imp.Alias), imp.P, nil})
	}
	for _, decl := range file.Decls {
		d := decl
		switch n := d.(type) {
		case *PrimitiveDecl:
			desc := "Primitive " + n.N
			if n.Kind != nil {
				desc += ": " + printKind(n.Kind)
			}
			entries = append(entries, entry{desc, n.P, nil})
		case *AliasDecl:
			entries = append(entries, entry{"Alias " + n.N, n.P, func(prefix string) {
				for i, p := range n.Params {
					desc := "Param " + p.N
					if p.Kind != nil {
						desc += ": " + printKind(p.Kind)
					}
					leaf(&b, prefix, i == len(n.Params)-1 && n.Target == nil, desc, p.P)
				}
				if n.Target != nil {
					leaf(&b, prefix, true, "Target "+printTypeRef(n.Target), n.Target.P)
				}
			}})
		}
	}

	for i, e := range entries {
		last := i == len(entries)-1
		leaf(&b, "", last, e.desc, e.pos)
		if e.kids != nil {
			e.kids(branchPad(last))
		}
	}

	return b.String()
}

func leaf(b *strings.Builder, prefix string, last bool, desc string, pos Position) {
	fmt.Fprintf(b, "%s%s%s  %s\n", prefix, branchMark(last), desc, pos)
}

func branchMark(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

func branchPad(last bool) string {
	if last {
		return "    "
	}
	return "│   "
}
