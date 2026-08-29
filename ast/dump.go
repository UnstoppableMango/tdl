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
				kids := params(n.Params)
				if n.Target != nil {
					kids = append(kids, child{"Target " + printTypeRef(n.Target), n.Target.P})
				}
				writeChildren(&b, prefix, kids)
			}})

		case *NewtypeDecl:
			entries = append(entries, entry{"Newtype " + n.N, n.P, func(prefix string) {
				kids := params(n.Params)
				if n.Base != nil {
					kids = append(kids, child{"Base " + printTypeRef(n.Base), n.Base.P})
				}
				kids = append(kids, constraints(n.Constraints)...)
				writeChildren(&b, prefix, kids)
			}})

		case *StructDecl:
			desc := strings.ToUpper(n.Keyword[:1]) + n.Keyword[1:] + " " + n.N
			entries = append(entries, entry{desc, n.P, func(prefix string) {
				kids := params(n.Params)
				for _, r := range n.Conforms {
					kids = append(kids, child{"Conforms " + r.N, r.P})
				}
				kids = append(kids, memberChildren(n.Members)...)
				writeChildren(&b, prefix, kids)
			}})

		case *EnumDecl:
			entries = append(entries, entry{"Enum " + n.N, n.P, func(prefix string) {
				kids := params(n.Params)
				for _, v := range n.Variants {
					desc := "Variant " + v.N
					if len(v.Fields) > 0 {
						desc += fmt.Sprintf(" (%d fields)", len(v.Fields))
					}
					kids = append(kids, child{desc, v.P})
				}
				writeChildren(&b, prefix, kids)
			}})

		case *ClassDecl:
			entries = append(entries, entry{"Class " + n.N, n.P, func(prefix string) {
				kids := params(n.Params)
				for _, d := range n.FunDeps {
					kids = append(kids, child{"FunDep " + strings.Join(d.From, " ") + " -> " + strings.Join(d.To, " "), d.P})
				}
				for _, r := range n.Conforms {
					kids = append(kids, child{"Requires " + r.N, r.P})
				}
				kids = append(kids, memberChildren(n.Members)...)
				writeChildren(&b, prefix, kids)
			}})

		case *InstanceDecl:
			desc := "Instance " + n.Class.N
			if n.For != nil {
				desc += " for " + printTypeRef(n.For)
			}
			entries = append(entries, entry{desc, n.P, func(prefix string) {
				kids := params(n.Params)
				for _, bind := range n.Binds {
					kids = append(kids, child{"Bind " + bind.N + " = " + printTypeRef(bind.Target), bind.P})
				}
				writeChildren(&b, prefix, kids)
			}})

		case *UnitDecl:
			desc := "Unit " + n.N
			if n.Expr != nil {
				desc += " = " + printUnitExpr(n.Expr)
			}
			entries = append(entries, entry{desc, n.P, nil})

		case *TargetDecl:
			entries = append(entries, entry{"Target " + n.N + " for " + n.For, n.P, func(prefix string) {
				writeChildren(&b, prefix, targetChildren(n.Entries))
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

// fieldSummary renders a field on one line, counting its constraints
// rather than expanding them.
func fieldSummary(f *Field) string {
	s := printFieldHead(f)
	if n := len(f.Constraints); n > 0 {
		s += fmt.Sprintf(" (%d constraints)", n)
	}
	if f.Default != nil {
		s += " = " + printLiteral(f.Default)
	}
	return s
}

type child struct {
	desc string
	pos  Position
}

func memberChildren(members []Member) []child {
	var kids []child
	for _, m := range members {
		switch mem := m.(type) {
		case *Include:
			kids = append(kids, child{"Include " + mem.Type.N, mem.P})
		case *KeyRequirement:
			kids = append(kids, child{"KeyRequirement", mem.P})
		case *AssocTypeReq:
			desc := "AssocType " + mem.N
			if mem.Kind != nil {
				desc += ": " + printKind(mem.Kind)
			}
			kids = append(kids, child{desc, mem.P})
		case *Field:
			kids = append(kids, child{"Field " + fieldSummary(mem), mem.P})
		}
	}
	return kids
}

func constraints(cs []*Constraint) []child {
	kids := make([]child, 0, len(cs))
	for _, c := range cs {
		kids = append(kids, child{"Constraint " + printConstraint(c), c.P})
	}
	return kids
}

func params(ps []*TypeParam) []child {
	kids := make([]child, 0, len(ps))
	for _, p := range ps {
		desc := "Param " + p.N
		if p.Kind != nil {
			desc += ": " + printKind(p.Kind)
		}
		kids = append(kids, child{desc, p.P})
	}
	return kids
}

// targetChildren flattens nested entries onto one level, prefixing each with
// the path that scopes it, so the dump stays one node per line.
func targetChildren(entries []*TargetEntry) []child {
	var kids []child
	for _, e := range entries {
		switch {
		case e.Entries != nil:
			for _, k := range targetChildren(e.Entries) {
				kids = append(kids, child{e.Path + "." + strings.TrimPrefix(k.desc, "Entry "), k.pos})
			}
		case e.Path != "":
			kids = append(kids, child{"Entry " + e.Path + " => " + printDirective(e.Directive), e.P})
		default:
			kids = append(kids, child{"Entry " + printDirective(e.Directive), e.P})
		}
	}
	return kids
}

func writeChildren(b *strings.Builder, prefix string, kids []child) {
	for i, k := range kids {
		leaf(b, prefix, i == len(kids)-1, k.desc, k.pos)
	}
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
