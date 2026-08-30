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

func (p *parser) parseNewtypeDecl(head ast.DeclHead) *ast.NewtypeDecl {
	head.P = p.cur.Pos
	p.next() // 'type'

	d := &ast.NewtypeDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if !p.expect(lex.COLON) {
		p.syncTop()
		return d
	}
	d.Base = p.parseTypeRef()
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}
	if p.at(lex.WHERE) {
		d.Constraints = p.parseConstraintBlock()
	}
	return d
}

func (p *parser) parseStructDecl(head ast.DeclHead) *ast.StructDecl {
	head.P = p.cur.Pos
	d := &ast.StructDecl{DeclHead: head, Keyword: p.cur.Text}
	p.next() // 'entity', 'value', or 'mixin'

	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if p.at(lex.COLON) {
		d.Conforms = p.parseConforms()
	}
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}
	d.Members = p.parseBody()
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
		d.Conforms = p.parseConforms()
	}
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}

	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return d
	}
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		d.Variants = append(d.Variants, p.parseVariant())
		if p.cur == before {
			p.next() // no progress: drop the offending token
		}
	}
	p.expect(lex.RBRACE)
	return d
}

func (p *parser) parseVariant() *ast.Variant {
	v := &ast.Variant{Doc: p.parseDoc(), P: p.cur.Pos}
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
		p.expect(lex.RBRACE)
	}
	return v
}

// parseConforms parses `":" ClassRef { "," ClassRef }`.
func (p *parser) parseConforms() []*ast.ClassRef {
	p.next() // ':'
	return p.parseClassRefList()
}

// parseClassRefs parses a `requires` clause.
func (p *parser) parseClassRefs() []*ast.ClassRef {
	p.next() // 'requires'
	return p.parseClassRefList()
}

func (p *parser) parseClassRefList() []*ast.ClassRef {
	var refs []*ast.ClassRef
	for {
		refs = append(refs, p.parseClassRef())
		if !p.accept(lex.COMMA) {
			return refs
		}
	}
}

func (p *parser) parseClassRef() *ast.ClassRef {
	ref := &ast.ClassRef{P: p.cur.Pos, N: p.expectIdent()}
	if p.at(lex.DOT) {
		p.next()
		ref.Qualifier, ref.N = ref.N, p.expectIdent()
	}
	if p.at(lex.LT) {
		ref.Args = p.parseTypeArgs()
	}
	return ref
}

func (p *parser) parseBody() []ast.Member {
	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return nil
	}

	var members []ast.Member
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur

		if p.at(lex.INCLUDE) && p.peek.Kind != lex.COLON {
			pos := p.cur.Pos
			p.next()
			members = append(members, &ast.Include{P: pos, Type: p.parseClassRef()})
		} else {
			members = append(members, p.parseField())
		}

		if p.cur == before {
			p.next()
		}
	}
	p.expect(lex.RBRACE)
	return members
}

// parseFields parses the field list inside an enum variant payload.
func (p *parser) parseFields() []*ast.Field {
	var fields []*ast.Field
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		fields = append(fields, p.parseField())
		if p.cur == before {
			p.next()
		}
	}
	return fields
}

func (p *parser) parseField() *ast.Field {
	if p.at(lex.COMMA) {
		p.errs.add(p.cur.Pos, "unexpected comma: commas are not separators inside a block")
		p.next()
		return &ast.Field{P: p.cur.Pos}
	}

	f := &ast.Field{Doc: p.parseDoc(), P: p.cur.Pos}

	// `key` and `deprecated` are contextual, so a field may be named either.
	// A modifier is a modifier only when another token follows it before the
	// colon.
	for {
		if p.atContextual("key") && p.peek.Kind != lex.COLON {
			f.Key = true
			p.next()
			continue
		}
		if p.atContextual("deprecated") && p.peek.Kind != lex.COLON {
			f.Dep = p.parseDeprecated()
			continue
		}
		break
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
	if p.at(lex.WHERE) {
		f.Constraints = p.parseConstraintBlock()
	}
	if p.accept(lex.EQUAL) {
		f.Default = p.parseLiteral()
	}
	return f
}

// expectFieldName reads a field name, accepting a reserved keyword when a
// colon follows it. `value`, `type`, and `unit` are ordinary words in a
// domain model, and the prelude's own `Option<T>` has a field named
// `value`. One token of lookahead settles it: `include Foo` is an include,
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
