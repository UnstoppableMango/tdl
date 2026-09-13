package parser

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// parseDeprecated parses `deprecated [ "(" string ")" ]`. `deprecated` is a
// contextual keyword, so the caller has already checked the identifier text.
func (p *parser) parseDeprecated() *ast.Deprecation {
	dep := &ast.Deprecation{P: p.cur.Pos}
	p.next() // 'deprecated'

	if p.accept(lex.LPAREN) {
		if p.at(lex.STRING) {
			dep.Reason = p.cur.Text
			p.next()
		} else {
			p.errs.add(p.cur.Pos, "expected a deprecation reason string, got %s", p.cur.Kind)
		}
		p.expect(lex.RPAREN)
	}
	return dep
}

// atContextual reports whether the current token is the contextual keyword
// word. Modifiers and constraint names are not reserved, so they arrive as
// ordinary identifiers and are recognized by position.
func (p *parser) atContextual(word string) bool {
	return p.cur.Kind == lex.IDENT && p.cur.Text == word
}

// parseTypeDecl parses both forms `type` introduces. A body makes it a
// domain type and the colon list the classes it conforms to; without one it
// is a newtype and the colon names its base. A newtype's constraints open
// with `where`, so a `{` here can only be a body.
func (p *parser) parseTypeDecl(head ast.DeclHead) ast.Decl {
	head.P = p.cur.Pos
	p.next() // 'type'

	head.N = p.expectIdent()
	var params []*ast.TypeParam
	if p.at(lex.LT) {
		params = p.parseTypeParams()
	}
	var refs []*ast.TypeRef
	if p.accept(lex.COLON) {
		refs = append(refs, p.parseTypeRef())
		for p.accept(lex.COMMA) {
			refs = append(refs, p.parseTypeRef())
		}
	}
	var requires []*ast.ClassRef
	if p.at(lex.REQUIRES) {
		requires = p.parseClassRefs()
	}

	if p.at(lex.LBRACE) || len(refs) == 0 {
		d := &ast.StructDecl{DeclHead: head, Keyword: "type", Params: params, Requires: requires}
		for _, r := range refs {
			d.Conforms = append(d.Conforms, p.classRefOf(r))
		}
		d.Members, d.End = p.parseBody()
		return d
	}

	if len(refs) > 1 {
		p.errs.add(refs[1].P, "a newtype has one base; a conformance list needs a body")
	}
	d := &ast.NewtypeDecl{DeclHead: head, Params: params, Base: refs[0], Requires: requires}
	if p.at(lex.WHERE) {
		d.Constraints, d.End = p.parseConstraintBlock()
	}
	return d
}

// classRefOf reads a type reference parsed before a body said it was a
// conformance list. A class is a name with arguments, so any other form is
// an error rather than a class.
func (p *parser) classRefOf(t *ast.TypeRef) *ast.ClassRef {
	if t.N == "" || t.Optional || t.Nullable {
		p.errs.add(t.P, "expected a class, got a type")
	}
	return &ast.ClassRef{P: t.P, Qualifier: t.Qualifier, N: t.N, Args: t.Args}
}

func (p *parser) parseStructDecl(head ast.DeclHead) *ast.StructDecl {
	head.P = p.cur.Pos
	d := &ast.StructDecl{DeclHead: head, Keyword: p.cur.Text}
	p.next() // 'mixin'

	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if p.at(lex.COLON) {
		d.Conforms = p.parseClassRefs()
	}
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}
	d.Members, d.End = p.parseBody()
	return d
}

func (p *parser) parseEnumDecl(head ast.DeclHead) *ast.EnumDecl {
	head.P = p.cur.Pos
	p.next() // 'enum'

	d := &ast.EnumDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if p.at(lex.COLON) {
		d.Conforms = p.parseClassRefs()
	}
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}

	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return d
	}
	p.untilRbrace(func() { d.Variants = append(d.Variants, p.parseVariant()) })
	d.End = p.expectRbrace()
	return d
}

func (p *parser) parseVariant() *ast.Variant {
	doc := p.parseDoc()
	v := &ast.Variant{DeclHead: ast.DeclHead{Doc: doc, P: p.cur.Pos}}
	if p.atContextual("deprecated") {
		v.Dep = p.parseDeprecated()
		v.P = p.cur.Pos
	}

	v.N = p.expectIdent()
	if p.at(lex.EQUAL) {
		p.errs.add(p.cur.Pos, "enum variants carry fields, not values")
		p.next()
		p.next() // the value
		return v
	}
	if p.accept(lex.LBRACE) {
		v.Fields = p.parseFields()
		v.End = p.expectRbrace()
	}
	return v
}

// parseClassRefs parses `ClassRef { "," ClassRef }` after the `:` of a
// conformance list or the `requires` of a constraint clause, consuming
// whichever introduced it.
func (p *parser) parseClassRefs() []*ast.ClassRef {
	p.next() // ':' or 'requires'

	var refs []*ast.ClassRef
	for {
		refs = append(refs, p.parseClassRef())
		if !p.accept(lex.COMMA) {
			return refs
		}
	}
}

func (p *parser) parseClassRef() *ast.ClassRef {
	ref := &ast.ClassRef{P: p.cur.Pos}
	ref.Qualifier, ref.N = p.parseQualified()
	if p.at(lex.LT) {
		ref.Args = p.parseTypeArgs()
	}
	return ref
}

func (p *parser) parseBody() ([]ast.Member, ast.Position) {
	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return nil, ast.Position{}
	}

	var members []ast.Member
	p.untilRbrace(func() {
		if p.at(lex.INCLUDE) && p.peek.Kind != lex.COLON {
			pos := p.cur.Pos
			p.next()
			members = append(members, &ast.Include{P: pos, Type: p.parseClassRef()})
		} else {
			members = append(members, p.parseField())
		}
	})
	return members, p.expectRbrace()
}

// parseFields parses the field list inside an enum variant payload.
func (p *parser) parseFields() []*ast.Field {
	var fields []*ast.Field
	p.untilRbrace(func() { fields = append(fields, p.parseField()) })
	return fields
}

func (p *parser) parseField() *ast.Field {
	if p.at(lex.COMMA) {
		p.errs.add(p.cur.Pos, "unexpected comma: commas are not separators inside a block")
		p.next()
		return &ast.Field{DeclHead: ast.DeclHead{P: p.cur.Pos}}
	}

	doc := p.parseDoc()
	f := &ast.Field{DeclHead: ast.DeclHead{Doc: doc, P: p.cur.Pos}}

	// `deprecated` is contextual, so a field may be named it. It is a
	// modifier only when another token follows it before the colon.
	if p.atContextual("deprecated") && p.peek.Kind != lex.COLON {
		f.Dep = p.parseDeprecated()
	}

	f.P = p.cur.Pos
	f.N = p.expectFieldName()
	if !p.expect(lex.COLON) {
		p.syncMember()
		return f
	}
	f.Type = p.parseTypeRef()

	// `owned` is contextual too: a following field named `owned` would
	// otherwise be swallowed as this field's relationship marker.
	if p.atContextual("owned") && p.peek.Kind != lex.COLON {
		f.Owned = true
		p.next()
	}
	// `where` is reserved, but a reserved word before a colon is a field
	// name, so a following field named `where` is not this field's
	// constraint block. The modifiers above take the same lookahead.
	if p.at(lex.WHERE) && p.peek.Kind != lex.COLON {
		f.Constraints, f.End = p.parseConstraintBlock()
	}
	if p.accept(lex.EQUAL) {
		f.Default = p.parseLiteral()
	}
	return f
}

// expectFieldName reads a field name, accepting a reserved keyword when a
// colon follows it. `type`, `unit`, and `include` are ordinary words in a
// domain model. One token of lookahead settles it: `include Foo` is an include,
// `include: Foo` is a field.
func (p *parser) expectFieldName() string {
	if p.cur.Kind != lex.IDENT && (!lex.IsKeyword(p.cur.Text) || p.peek.Kind != lex.COLON) {
		p.errs.add(p.cur.Pos, "expected a field name, got %s", p.cur.Kind)
		return ""
	}
	name := p.cur.Text
	p.next()
	return name
}

// syncMember skips to the end of the enclosing body or the start of
// something that plausibly begins the next member.
func (p *parser) syncMember() {
	for {
		switch p.cur.Kind {
		case lex.RBRACE, lex.EOF, lex.DOC, lex.INCLUDE:
			return
		}
		if p.cur.Kind == lex.IDENT && p.peek.Kind == lex.COLON {
			return
		}
		p.next()
	}
}

func (p *parser) parseLiteral() *ast.Literal {
	lit := &ast.Literal{P: p.cur.Pos, Text: p.cur.Text}

	switch p.cur.Kind {
	case lex.STRING:
		lit.Kind = ast.LitString
	case lex.INT:
		lit.Kind = ast.LitInt
	case lex.FLOAT:
		lit.Kind = ast.LitFloat
	case lex.TRUE, lex.FALSE:
		lit.Kind = ast.LitBool
	case lex.LBRACK:
		lit.Kind = ast.LitList
		lit.Text = ""
		p.next()
		for !p.at(lex.RBRACK) && !p.at(lex.EOF) {
			lit.Items = append(lit.Items, p.parseLiteral())
			if !p.accept(lex.COMMA) {
				break
			}
		}
		p.expect(lex.RBRACK)
		return lit
	case lex.IDENT:
		// A name denotes an enum variant; the resolver checks it against the
		// field's type.
		lit.Kind = ast.LitName
		lit.Text = p.parseDottedIdent()
		return lit
	default:
		p.errs.add(p.cur.Pos, "expected a literal, got %s", p.cur.Kind)
		return lit
	}

	p.next()
	return lit
}
