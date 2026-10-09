package ast

import (
	"math"
	"strconv"
	"strings"
)

// printer renders a file, placing ordinary comments by position: on their
// own line before the next item, or folded onto the end of the line they
// were written on. The cursor only moves forward, so each comment is written
// once.
type printer struct {
	b        strings.Builder
	comments []*Comment
	i        int
}

func (p *printer) pending() *Comment {
	if p.i < len(p.comments) {
		return p.comments[p.i]
	}
	return nil
}

// before reports whether an unwritten comment precedes pos. Asked against
// a block's end, it reports whether the block holds a comment.
func (p *printer) before(pos Position) bool {
	c := p.pending()
	return c != nil && c.P.Offset < pos.Offset
}

// take returns the comments before pos without writing them.
func (p *printer) take(pos Position) []*Comment {
	var out []*Comment
	for p.before(pos) {
		out = append(out, p.comments[p.i])
		p.i++
	}
	return out
}

// flush writes every comment before pos on a line of its own at indent.
func (p *printer) flush(indent string, pos Position) {
	for p.before(pos) {
		p.writeComment(indent, p.comments[p.i])
		p.i++
	}
}

// writeLead interleaves collected comments with doc comment lines by offset.
func (p *printer) writeLead(indent string, lead []*Comment, doc []string, docPos []Position) {
	i := 0
	for j, line := range doc {
		k := i
		for k < len(lead) && lead[k].P.Offset < docPos[j].Offset {
			k++
		}
		p.writeComments(indent, lead[i:k])
		i = k
		p.writeDoc(indent, line)
	}
	p.writeComments(indent, lead[i:])
}

// separated reports whether a blank line separates lines prev and next.
func separated(prev, next int) bool { return next > prev+1 }

// writeComments writes comments, keeping blank lines between them.
func (p *printer) writeComments(indent string, cs []*Comment) {
	for i, c := range cs {
		if i > 0 && separated(cs[i-1].P.Line, c.P.Line) {
			p.b.WriteString("\n")
		}
		p.writeComment(indent, c)
	}
}

// lead writes the comments and doc comment in front of an item at pos.
func (p *printer) lead(indent string, pos Position, doc []string, docPos []Position) {
	p.writeLead(indent, p.take(pos), doc, docPos)
}

func (p *printer) writeComment(indent string, c *Comment) {
	p.b.WriteString(strings.TrimRight(indent+"// "+c.Text, " ") + "\n")
}

func (p *printer) writeDoc(indent, line string) {
	p.b.WriteString(strings.TrimRight(indent+"/// "+line, " ") + "\n")
}

// trailing returns the comment on line and before until, formatted to fold
// onto the line's end, or "". The bound keeps a comment after a later item
// on the same line, such as the closing brace of a one-line block the
// formatter opens up, from folding onto this one.
func (p *printer) trailing(line int, until Position) string {
	c := p.pending()
	if c == nil || c.P.Line != line || c.P.Offset >= until.Offset {
		return ""
	}
	p.i++
	return strings.TrimRight("  // "+c.Text, " ")
}

// line writes s, folding in a trailing comment written before until.
func (p *printer) line(s string, srcLine int, until Position) {
	p.b.WriteString(s + p.trailing(srcLine, until) + "\n")
}

// firstPos is the position of a block's first item, or end when empty. It
// bounds the comment folded onto the opening brace, so a comment after the
// first item of a one-line block stays with that item.
func firstPos[T any](items []T, pos func(T) Position, end Position) Position {
	if len(items) == 0 {
		return end
	}
	return pos(items[0])
}

// anywhere is the trailing-comment bound for a declaration's last line.
var anywhere = Position{Offset: math.MaxInt}

// render runs f on a sub-printer and returns its output, advancing this
// cursor past the comments f consumed. It lets a caller measure output
// before placing it.
func (p *printer) render(f func(*printer)) string {
	sub := &printer{comments: p.comments, i: p.i}
	f(sub)
	p.i = sub.i
	return sub.b.String()
}

// Fprint renders file as canonical TDL source, the output of `tdl fmt`.
// Layout depends only on the tree and its comments, never on the input's
// whitespace, so formatting is idempotent.
func Fprint(file *File) string {
	p := &printer{comments: file.Comments}

	if pkg := file.Package; pkg != nil {
		p.lead("", pkg.P, pkg.Doc, pkg.DocP)
		p.line("package "+pkg.Path, pkg.P.Line, anywhere)
	}

	if len(file.Imports) > 0 {
		if p.b.Len() > 0 {
			p.b.WriteString("\n")
		}
		for _, imp := range file.Imports {
			p.lead("", imp.P, imp.Doc, imp.DocP)
			p.line("import "+quote(imp.Path)+" as "+imp.Alias, imp.P.Line, anywhere)
		}
	}

	prevMultiline := true // force a blank line after the header, if any
	for _, decl := range file.Decls {
		// Leading comments are collected first: they decide the blank line.
		lead := p.take(decl.Pos())
		text := p.render(func(sub *printer) { sub.decl(decl) })
		multiline := strings.Count(text, "\n") > 1
		head := decl.Head()
		annotated := len(head.Doc) > 0 || head.Dep != nil || len(lead) > 0

		// Consecutive one-line declarations group together; anything with a
		// body, doc comment, deprecation, or comment gets a blank line.
		if p.b.Len() > 0 && (multiline || prevMultiline || annotated) {
			p.b.WriteString("\n")
		}

		p.writeLead("", lead, head.Doc, head.DocP)
		// A blank line between a comment and the declaration survives.
		if n := len(lead); n > 0 && (len(head.Doc) == 0 || lead[n-1].P.Offset > head.DocP[len(head.DocP)-1].Offset) {
			next := decl.Pos().Line
			if head.Dep != nil {
				next = head.Dep.P.Line
			}
			if separated(lead[n-1].P.Line, next) {
				p.b.WriteString("\n")
			}
		}
		if head.Dep != nil {
			p.b.WriteString(printDeprecated(head.Dep) + "\n")
		}
		p.b.WriteString(text)
		prevMultiline = multiline || annotated
	}

	if p.before(file.End) {
		if p.b.Len() > 0 {
			p.b.WriteString("\n")
		}
		p.flush("", file.End)
	}

	return p.b.String()
}

// PrintDecl renders one declaration in canonical form, without its doc
// comment, deprecation, or surrounding comments, as a hover shows it.
func PrintDecl(decl Decl) string {
	p := &printer{}
	p.decl(decl)
	return p.b.String()
}

func (p *printer) decl(decl Decl) {
	switch d := decl.(type) {
	case *PrimitiveDecl:
		s := "primitive " + d.N
		if d.Kind != nil {
			s += ": " + printKind(d.Kind)
		}
		p.line(s, d.P.Line, anywhere)

	case *AliasDecl:
		p.line("alias "+d.N+printParams(d.Params)+" = "+printTypeRef(d.Target), d.P.Line, anywhere)

	case *NewtypeDecl:
		head := "type " + d.N + printParams(d.Params) + ": " + printTypeRef(d.Base) + printRequires(d.Requires)
		p.constrained(head, d.Constraints, "", d.P.Line, d.End)

	case *StructDecl:
		p.b.WriteString(d.Keyword + " " + d.N + printParams(d.Params) +
			printConforms(d.Conforms) + printRequires(d.Requires))
		p.members(d.Members, d.P.Line, d.End)

	case *EnumDecl:
		p.b.WriteString("enum " + d.N + printParams(d.Params) +
			printConforms(d.Conforms) + printRequires(d.Requires))
		p.variants(d.Variants, d.P.Line, d.End)

	case *ClassDecl:
		p.b.WriteString("class " + d.N + printParams(d.Params) + printFunDeps(d.FunDeps) +
			printConforms(d.Conforms) + printRequires(d.Requires))
		p.members(d.Members, d.P.Line, d.End)

	case *InstanceDecl:
		p.instance(d)

	case *UnitDecl:
		s := "unit " + d.N
		if d.Expr != nil {
			s += " = " + PrintUnitExpr(d.Expr)
		}
		p.line(s, d.P.Line, anywhere)

	case *TargetDecl:
		p.b.WriteString("target " + d.N + " for " + d.For)
		p.entries(d.Entries, "", d.P.Line, d.End, anywhere)
	}
}

// constrained writes head and its `where` block. A trailing comment folds
// onto the construct's last line.
func (p *printer) constrained(head string, cs []*Constraint, indent string, headLine int, end Position) {
	block := p.constraints(cs, indent, end)
	line := headLine
	if strings.Contains(block, "\n") {
		line = end.Line
	}
	p.line(head+block, line, anywhere)
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

// instance keeps the written form: `instance C for T` is not rewritten to
// `instance C<T>`.
func (p *printer) instance(d *InstanceDecl) {
	s := "instance "
	if len(d.Params) > 0 {
		s += printParams(d.Params) + " "
	}
	s += printClassRefs([]*ClassRef{d.Class})
	if d.For != nil {
		s += " for " + printTypeRef(d.For)
	}
	s += printRequires(d.Requires)

	// An empty bind block without comments is dropped.
	if len(d.Binds) == 0 && !p.before(d.End) {
		p.line(s, d.P.Line, anywhere)
		return
	}

	p.b.WriteString(s + " {" + p.trailing(d.P.Line, firstPos(d.Binds, func(b *AssocTypeBind) Position { return b.P }, d.End)) + "\n")
	for _, bind := range d.Binds {
		p.flush("  ", bind.P)
		p.line("  type "+bind.N+" = "+printTypeRef(bind.Target), bind.P.Line, d.End)
	}
	p.flush("  ", d.End)
	p.line("}", d.End.Line, anywhere)
}

// members writes a declaration body. An empty one without comments
// collapses to `{ }`.
func (p *printer) members(members []Member, headLine int, end Position) {
	if len(members) == 0 && !p.before(end) {
		p.line(" { }", headLine, anywhere)
		return
	}

	p.b.WriteString(" {" + p.trailing(headLine, firstPos(members, Member.Pos, end)) + "\n")
	for _, m := range members {
		switch n := m.(type) {
		case *Include:
			p.flush("  ", n.P)
			p.line("  include "+printClassRefs([]*ClassRef{n.Type}), n.P.Line, end)
		case *AssocTypeReq:
			p.lead("  ", n.P, n.Doc, n.DocP)
			s := "  type " + n.N
			if n.Kind != nil {
				s += ": " + printKind(n.Kind)
			}
			p.line(s, n.P.Line, end)
		case *Field:
			p.lead("  ", n.P, n.Doc, n.DocP)
			p.field(n, "  ", end)
		}
	}
	p.flush("  ", end)
	p.line("}", end.Line, anywhere)
}

func (p *printer) field(f *Field, indent string, until Position) {
	tail := p.fieldTail(f, indent)
	line := f.P.Line
	if strings.Contains(tail, "\n") {
		line = f.End.Line
	}
	p.line(indent+tail, line, until)
}

// fieldTail renders a field without indent or trailing comment. The
// constraint block may span lines, so the caller places the comment.
func (p *printer) fieldTail(f *Field, indent string) string {
	s := printFieldHead(f) + p.constraints(f.Constraints, indent, f.End)
	if f.Default != nil {
		s += " = " + printLiteral(f.Default)
	}
	return s
}

// PrintField renders a field as one struct member, with its deprecation and
// without its doc comment or indent. A constraint block of two or more
// constraints spans lines.
func PrintField(f *Field) string {
	return (&printer{}).fieldTail(f, "")
}

// printFieldHead renders a field up to its constraint block. [Dump] uses
// it to keep a field on one line.
func printFieldHead(f *Field) string {
	var s string
	if f.Dep != nil {
		s += printDeprecated(f.Dep) + " "
	}
	s += f.N + ": " + printTypeRef(f.Type)
	if f.Owned {
		s += " owned"
	}
	return s
}

func (p *printer) variants(variants []*Variant, headLine int, end Position) {
	if len(variants) == 0 && !p.before(end) {
		p.line(" { }", headLine, anywhere)
		return
	}

	// A block stays on one line when it fits the column limit and holds no
	// comment.
	inline := !p.before(end)
	oneLine := " {"
	if inline {
		for _, v := range variants {
			if len(v.Fields) > 0 || len(v.Doc) > 0 || v.Dep != nil {
				inline = false
				break
			}
			oneLine += " " + v.N
		}
	}
	if inline && len(oneLine)+2 <= columnLimit {
		p.line(oneLine+" }", headLine, anywhere)
		return
	}

	p.b.WriteString(" {" + p.trailing(headLine, firstPos(variants, func(v *Variant) Position { return v.P }, end)) + "\n")
	for _, v := range variants {
		p.lead("  ", v.P, v.Doc, v.DocP)

		s := "  "
		if v.Dep != nil {
			s += printDeprecated(v.Dep) + " "
		}
		s += v.N

		if p.expands(v) {
			p.b.WriteString(s + " {" + p.trailing(v.P.Line, firstPos(v.Fields, func(f *Field) Position { return f.P }, v.End)) + "\n")
			for _, f := range v.Fields {
				p.lead("    ", f.P, f.Doc, f.DocP)
				p.field(f, "    ", v.End)
			}
			p.flush("    ", v.End)
			p.line("  }", v.End.Line, end)
			continue
		}

		line := v.P.Line
		if len(v.Fields) > 0 {
			parts := make([]string, len(v.Fields))
			for i, f := range v.Fields {
				parts[i] = p.fieldTail(f, "  ")
			}
			s += " { " + strings.Join(parts, " ") + " }"
			if v.End.Line != 0 {
				line = v.End.Line
			}
		}
		p.line(s, line, end)
	}
	p.flush("  ", end)
	p.line("}", end.Line, anywhere)
}

// expands reports whether a variant's payload needs several lines: it
// holds a comment, a documented field, or a multi-line constraint block.
func (p *printer) expands(v *Variant) bool {
	if v.End.Line == 0 {
		return false
	}
	if p.before(v.End) {
		return true
	}
	// With no comment in the payload, measuring a tail consumes nothing.
	for _, f := range v.Fields {
		if len(f.Doc) > 0 || strings.Contains(p.fieldTail(f, "  "), "\n") {
			return true
		}
	}
	return false
}

// entries writes a target block ending at end, nested in one ending at
// until.
func (p *printer) entries(entries []*TargetEntry, indent string, headLine int, end, until Position) {
	if len(entries) == 0 && !p.before(end) {
		p.line(" { }", headLine, until)
		return
	}

	p.b.WriteString(" {" + p.trailing(headLine, firstPos(entries, func(e *TargetEntry) Position { return e.P }, end)) + "\n")
	inner := indent + "  "
	for _, e := range entries {
		p.flush(inner, e.P)
		switch {
		case e.Entries != nil:
			p.b.WriteString(inner + e.Path)
			p.entries(e.Entries, inner, e.P.Line, e.End, end)
		case e.Path != "":
			p.line(inner+e.Path+" => "+printDirective(e.Directive), e.P.Line, end)
		default:
			p.line(inner+printDirective(e.Directive), e.P.Line, end)
		}
	}
	p.flush(inner, end)
	p.line(indent+"}", end.Line, until)
}

func printDirective(d *Directive) string { return printCall(d.N, d.Args) }

func printConstraint(c *Constraint) string { return printCall(c.N, c.Args) }

// printCall renders a directive or constraint: `min(0)`, `tag("x")`.
func printCall(name string, args []*Literal) string {
	if len(args) == 0 {
		return name
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = printLiteral(a)
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
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

// constraints renders a `where { ... }` block. A single constraint stays
// on one line when it fits the column limit; two or more always expand.
func (p *printer) constraints(cs []*Constraint, indent string, end Position) string {
	if len(cs) == 0 {
		return ""
	}

	// A comment inside forces the expanded form.
	if len(cs) == 1 && !p.before(end) {
		if line := " where { " + printConstraint(cs[0]) + " }"; len(indent)+len(line) <= columnLimit {
			return line
		}
	}

	return p.render(func(sub *printer) {
		sub.b.WriteString(" where {\n")
		for _, c := range cs {
			sub.flush(indent+"  ", c.P)
			sub.line(indent+"  "+printConstraint(c), c.P.Line, end)
		}
		sub.flush(indent+"  ", end)
		sub.b.WriteString(indent + "}")
	})
}

// columnLimit is the width a one-line block must fit within.
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
			parts[i] = PrintUnitExpr(a.Unit)
		} else {
			parts[i] = printTypeRef(a.Type)
		}
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// PrintUnitExpr renders a unit expression without spaces around its
// operators: `kg*m/s^2`. Lowering uses it to record a unit's spelling.
func PrintUnitExpr(e *UnitExpr) string {
	var b strings.Builder
	for _, t := range e.Terms {
		b.WriteString(t.Op)
		if t.Paren != nil {
			b.WriteString("(" + PrintUnitExpr(t.Paren) + ")")
		} else {
			b.WriteString(t.N)
		}
		if t.Exp != 1 {
			b.WriteString("^" + strconv.Itoa(t.Exp))
		}
	}
	return b.String()
}

// PrintTypeRef renders a type reference as the formatter writes it.
func PrintTypeRef(t *TypeRef) string { return printTypeRef(t) }

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
