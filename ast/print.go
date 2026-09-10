package ast

import (
	"math"
	"strconv"
	"strings"
)

// printer renders a file, placing as it goes the ordinary comments the
// parser collected.
//
// A comment is attached to no node, so it is placed by position: written on
// its own line before the item that follows it, or folded onto the end of
// the line it was written on. The cursor only moves forward, which is what
// says each comment is written exactly once.
type printer struct {
	b        strings.Builder
	comments []*Comment
	i        int
}

// pending is the next comment not yet written, or nil.
func (p *printer) pending() *Comment {
	if p.i < len(p.comments) {
		return p.comments[p.i]
	}
	return nil
}

// before reports whether a comment is still waiting that was written before
// pos. Everything earlier has already been placed, so this also answers
// "is there a comment inside this block", asked against the block's end.
func (p *printer) before(pos Position) bool {
	c := p.pending()
	return c != nil && c.P.Offset < pos.Offset
}

// take collects the comments written before pos rather than writing them,
// for a caller that has to know they exist before it can lay them out.
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

func (p *printer) writeComment(indent string, c *Comment) {
	p.b.WriteString(strings.TrimRight(indent+"// "+c.Text, " ") + "\n")
}

// trailing returns the comment written on line and before until, folded
// onto the end of the line, or "" when there is none. A `//` comment runs
// to the end of its line, so one sharing a line with an item always follows
// it; the bound is what says it follows this item and not a later one on
// the same line. A block written on one line and opened up by the
// formatter would otherwise fold a comment after its closing brace onto
// its first line, so an item inside a block is bounded by the block's end.
func (p *printer) trailing(line int, until Position) string {
	c := p.pending()
	if c == nil || c.P.Line != line || c.P.Offset >= until.Offset {
		return ""
	}
	p.i++
	return strings.TrimRight("  // "+c.Text, " ")
}

// line writes one complete line, folding in a comment written on it before
// until.
func (p *printer) line(s string, srcLine int, until Position) {
	p.b.WriteString(s + p.trailing(srcLine, until) + "\n")
}

// firstPos is the position of a block's first item, or end when the block
// has none. It is what bounds a comment folded onto the opening brace: one
// written after the first item belongs to that item and not to the brace,
// which is what a block the source wrote on a single line looks like once
// the formatter opens it up.
func firstPos[T any](items []T, pos func(T) Position, end Position) Position {
	if len(items) == 0 {
		return end
	}
	return pos(items[0])
}

// anywhere is the bound for a line nothing follows on its own line: a
// declaration, or the brace closing one.
var anywhere = Position{Offset: math.MaxInt}

// render runs f on a printer of its own and returns what it wrote, moving
// this one's cursor over whatever f consumed. Fprint measures a
// declaration before deciding the blank line in front of it, and a
// constraint block is measured against the column limit the same way.
func (p *printer) render(f func(*printer)) string {
	sub := &printer{comments: p.comments, i: p.i}
	f(sub)
	p.i = sub.i
	return sub.b.String()
}

// Fprint renders file as canonical TDL source, the formatting produced by
// `tdl fmt`. It is idempotent: formatting canonical output changes nothing.
//
// Whitespace is insignificant in TDL, so the formatter owns layout
// entirely. Its decisions depend only on the tree and on where the comments
// sit in it, never on how the input was written, which is what makes
// idempotence hold.
func Fprint(file *File) string {
	p := &printer{comments: file.Comments}

	if file.Package != nil {
		p.flush("", file.Package.P)
		p.line("package "+file.Package.Path, file.Package.P.Line, anywhere)
	}

	if len(file.Imports) > 0 {
		if p.b.Len() > 0 {
			p.b.WriteString("\n")
		}
		for _, imp := range file.Imports {
			p.flush("", imp.P)
			writeDoc(&p.b, "", imp.Doc)
			p.line("import "+quote(imp.Path)+" as "+imp.Alias, imp.P.Line, anywhere)
		}
	}

	prevMultiline := true // force a blank line after the header, if any
	for _, decl := range file.Decls {
		// The comments in front of a declaration are collected rather than
		// written, because whether it has any is part of deciding the blank
		// line that goes before them.
		lead := p.take(decl.Pos())
		text := p.render(func(sub *printer) { sub.decl(decl) })
		multiline := strings.Count(text, "\n") > 1
		head := decl.Head()
		annotated := len(head.Doc) > 0 || head.Dep != nil || len(lead) > 0

		// Consecutive one-line declarations group together; anything with a
		// body, a doc comment, a deprecation, or a comment of its own gets a
		// blank line. The decision reads only the tree and the comments in
		// it, so formatting stays idempotent.
		if p.b.Len() > 0 && (multiline || prevMultiline || annotated) {
			p.b.WriteString("\n")
		}

		for _, c := range lead {
			p.writeComment("", c)
		}
		writeDoc(&p.b, "", head.Doc)
		if head.Dep != nil {
			p.b.WriteString(printDeprecated(head.Dep) + "\n")
		}
		p.b.WriteString(text)
		prevMultiline = multiline || annotated
	}

	// A comment after the last declaration has nothing to sit in front of.
	if p.before(file.End) {
		if p.b.Len() > 0 {
			p.b.WriteString("\n")
		}
		p.flush("", file.End)
	}

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

// constrained writes a head followed by its `where` block, if it has one.
// A trailing comment belongs to whichever line the construct ends on.
func (p *printer) constrained(head string, cs []*Constraint, indent string, headLine int, end Position) {
	block := p.constraints(cs, indent, end)
	line := headLine
	if strings.Contains(block, "\n") {
		line = end.Line
	}
	p.line(head+block, line, anywhere)
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

// instance keeps the form that was written. `instance C for T` is sugar
// for `instance C<T>`, and rewriting one into the other would lose what the
// author chose to say.
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

	// An empty bind block is dropped, unless a comment is sitting in it.
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

// members writes a declaration body. An empty one collapses to `{ }`,
// unless a comment is sitting in it.
func (p *printer) members(members []Member, headLine int, end Position) {
	if len(members) == 0 && !p.before(end) {
		p.line(" { }", headLine, anywhere)
		return
	}

	p.b.WriteString(" {" + p.trailing(headLine, firstPos(members, Member.Pos, end)) + "\n")
	for _, m := range members {
		p.flush("  ", m.Pos())
		switch n := m.(type) {
		case *Include:
			p.line("  include "+printClassRefs([]*ClassRef{n.Type}), n.P.Line, end)
		case *KeyRequirement:
			p.line("  key", n.P.Line, end)
		case *AssocTypeReq:
			writeDoc(&p.b, "  ", n.Doc)
			s := "  type " + n.N
			if n.Kind != nil {
				s += ": " + printKind(n.Kind)
			}
			p.line(s, n.P.Line, end)
		case *Field:
			writeDoc(&p.b, "  ", n.Doc)
			p.field(n, "  ", end)
		}
	}
	p.flush("  ", end)
	p.line("}", end.Line, anywhere)
}

// field writes one field inside a block ending at until.
func (p *printer) field(f *Field, indent string, until Position) {
	tail := p.fieldTail(f, indent)
	line := f.P.Line
	if strings.Contains(tail, "\n") {
		line = f.End.Line
	}
	p.line(indent+tail, line, until)
}

// fieldTail renders a field's head, its constraint block, and its default.
// The block is what may run to several lines, so the caller decides which
// line a trailing comment belongs to.
func (p *printer) fieldTail(f *Field, indent string) string {
	s := printFieldHead(f) + p.constraints(f.Constraints, indent, f.End)
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

func (p *printer) variants(variants []*Variant, headLine int, end Position) {
	if len(variants) == 0 && !p.before(end) {
		p.line(" { }", headLine, anywhere)
		return
	}

	// A block stays on one line when it fits the column limit. Whitespace is
	// insignificant, so this depends only on content: the same tree always
	// formats the same way, whatever the input looked like. A comment inside
	// forces the expanded form, since a one-line block has nowhere to put
	// one.
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
		p.flush("  ", v.P)
		writeDoc(&p.b, "  ", v.Doc)

		s := "  "
		if v.Dep != nil {
			s += printDeprecated(v.Dep) + " "
		}
		s += v.N

		if p.expands(v) {
			p.b.WriteString(s + " {" + p.trailing(v.P.Line, firstPos(v.Fields, func(f *Field) Position { return f.P }, v.End)) + "\n")
			for _, f := range v.Fields {
				p.flush("    ", f.P)
				writeDoc(&p.b, "    ", f.Doc)
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

// expands reports whether a variant's payload opens up rather than staying
// on the variant's line: it holds a comment, a field carrying a doc
// comment, or a field whose constraints run to several lines, none of
// which one line has room for. A payload holding none of these is written
// inline, and an empty one is dropped.
func (p *printer) expands(v *Variant) bool {
	if v.End.Line == 0 {
		return false
	}
	if p.before(v.End) {
		return true
	}
	// No comment sits in the payload, so rendering a tail here to measure
	// it consumes nothing, and rendering it again to write it is safe.
	for _, f := range v.Fields {
		if len(f.Doc) > 0 || strings.Contains(p.fieldTail(f, "  "), "\n") {
			return true
		}
	}
	return false
}

// entries writes a target block ending at end, inside a block ending at
// until when it is nested.
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

// printCall renders a name applied to literal arguments, which is what a
// directive and a constraint both are: `min(0)`, `tag("json:email")`.
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

// printConstraints renders a `where { ... }` block.
//
// A single constraint stays on one line when it fits the column limit; two
// or more always expand. One constraint is an aside on the thing it
// qualifies, several are a specification of it, and the layout says which.
func (p *printer) constraints(cs []*Constraint, indent string, end Position) string {
	if len(cs) == 0 {
		return ""
	}

	// A comment inside forces the expanded form, for the reason an enum
	// body expands: a one-line block has nowhere to put one.
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
			parts[i] = PrintUnitExpr(a.Unit)
		} else {
			parts[i] = printTypeRef(a.Type)
		}
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

// PrintUnitExpr renders a unit expression without spaces around its
// operators, the form the spec uses: `kg*m/s^2`.
//
// Exported because lowering records what a unit was written as beside what
// it reduces to, and reconstructing the text there would be a second
// printer to keep in step with this one.
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
