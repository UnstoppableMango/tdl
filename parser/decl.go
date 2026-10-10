package parser

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// parseDeprecated parses `deprecated [ "(" string ")" ]`; the caller has
// already matched the contextual keyword.
func (p *parser) parseDeprecated() *ast.Deprecation {
	dep := &ast.Deprecation{P: p.cur.Pos}
	defer func() { dep.E = p.endOf(dep.P) }()
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

// atContextual reports whether the current token is the identifier word.
func (p *parser) atContextual(word string) bool {
	return p.cur.Kind == lex.IDENT && p.cur.Text == word
}

// parseTypeDecl parses a domain type (a body, with the colon list naming
// classes) or a newtype (no body, with the colon naming its base).
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
		d.Members, d.Rbrace = p.parseBody()
		return d
	}

	if len(refs) > 1 {
		p.errs.add(refs[1].P, "a newtype has one base; a conformance list needs a body")
	}
	d := &ast.NewtypeDecl{DeclHead: head, Params: params, Base: refs[0], Requires: requires}
	if p.at(lex.WHERE) {
		d.Constraints, d.Rbrace = p.parseConstraintBlock()
	}
	return d
}

// classRefOf converts a type reference that turned out to be in a
// conformance list, rejecting forms a class cannot take.
func (p *parser) classRefOf(t *ast.TypeRef) *ast.ClassRef {
	if t.N == "" || t.Optional || t.Nullable {
		p.errs.add(t.P, "expected a class, got a type")
	}
	return &ast.ClassRef{P: t.P, E: t.E, Qualifier: t.Qualifier, N: t.N, Args: t.Args}
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
	d.Members, d.Rbrace = p.parseBody()
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
	d.Rbrace = p.expectRbrace()
	return d
}

func (p *parser) parseVariant() *ast.Variant {
	doc, docP := p.parseDoc()
	v := &ast.Variant{DeclHead: ast.DeclHead{Doc: doc, DocP: docP, P: p.cur.Pos}}
	defer func() { v.E = p.endOf(v.P) }()
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
		v.Rbrace = p.expectRbrace()
	}
	return v
}

// parseClassRefs parses `ClassRef { "," ClassRef }`, consuming the
// leading `:` or `requires`.
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
	ref.E = p.endOf(ref.P)
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
			inc := &ast.Include{P: pos, Type: p.parseClassRef()}
			inc.E = p.endOf(pos)
			members = append(members, inc)
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
		return &ast.Field{DeclHead: ast.DeclHead{P: p.cur.Pos, E: p.cur.Pos}}
	}

	doc, docP := p.parseDoc()
	f := &ast.Field{DeclHead: ast.DeclHead{Doc: doc, DocP: docP, P: p.cur.Pos}}
	defer func() { f.E = p.endOf(f.P) }()

	// `deprecated:` is a field name, not a modifier.
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

	// `owned:` is the next field, not this field's modifier.
	if p.atContextual("owned") && p.peek.Kind != lex.COLON {
		f.Owned = true
		p.next()
	}
	// Likewise `where:` is the next field, not a constraint block.
	if p.at(lex.WHERE) && p.peek.Kind != lex.COLON {
		f.Constraints, f.Rbrace = p.parseConstraintBlock()
	}
	if p.accept(lex.EQUAL) {
		f.Default = p.parseLiteral()
	}
	return f
}

// expectFieldName reads a field name, accepting a reserved keyword when a
// colon follows it: `include: Foo` is a field, `include Foo` an include.
func (p *parser) expectFieldName() string {
	if p.cur.Kind != lex.IDENT && (!lex.IsKeyword(p.cur.Text) || p.peek.Kind != lex.COLON) {
		p.errs.add(p.cur.Pos, "expected a field name, got %s", p.cur.Kind)
		return ""
	}
	name := p.cur.Text
	p.next()
	return name
}

// syncMember skips to the end of the body or a likely next member.
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
	defer func() { lit.E = p.endOf(lit.P) }()

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
		// An enum variant, checked against the field's type in lowering.
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
