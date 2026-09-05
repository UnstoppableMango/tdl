package parser

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

func (p *parser) parseClassDecl(head ast.DeclHead) *ast.ClassDecl {
	head.P = p.cur.Pos
	p.next() // 'class'

	d := &ast.ClassDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if p.at(lex.PIPE) {
		d.FunDeps = p.parseFunDeps()
	}
	if p.at(lex.COLON) {
		d.Conforms = p.parseConforms()
	}
	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}
	d.Members, d.End = p.parseClassBody()
	return d
}

// parseFunDeps parses `"|" FunDep { "," FunDep }`, where a FunDep is
// `ident { ident } "->" ident { ident }`.
func (p *parser) parseFunDeps() []*ast.FunDep {
	p.next() // '|'

	var deps []*ast.FunDep
	for {
		dep := &ast.FunDep{P: p.cur.Pos}
		for p.at(lex.IDENT) {
			dep.From = append(dep.From, p.cur.Text)
			p.next()
		}
		if !p.expect(lex.ARROW) {
			return append(deps, dep)
		}
		for p.at(lex.IDENT) {
			dep.To = append(dep.To, p.cur.Text)
			p.next()
		}
		deps = append(deps, dep)

		if !p.accept(lex.COMMA) {
			return deps
		}
	}
}

// parseClassBody parses the members a class may hold: fields, a bare `key`
// requirement, and associated type requirements.
func (p *parser) parseClassBody() ([]ast.Member, ast.Position) {
	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return nil, ast.Position{}
	}

	var members []ast.Member
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		doc := p.parseDoc()

		switch {
		// A class may not declare key fields, so `key` inside a class body is
		// always the requirement. It says an implementor must have identity,
		// never which field carries it. Without that rule `key` followed by a
		// field would be indistinguishable from `key field: T`, since
		// whitespace is insignificant.
		case p.atContextual("key") && p.peek.Kind != lex.COLON:
			members = append(members, &ast.KeyRequirement{P: p.cur.Pos})
			p.next()

		// `type Cursor` requires a type. `type: T` is a field named type.
		case p.at(lex.TYPE) && p.peek.Kind == lex.IDENT:
			req := &ast.AssocTypeReq{Doc: doc, P: p.cur.Pos}
			p.next()
			req.N = p.expectIdent()
			if p.accept(lex.COLON) {
				req.Kind = p.parseKind()
			}
			members = append(members, req)

		default:
			f := p.parseField()
			f.Doc = append(doc, f.Doc...)
			members = append(members, f)
		}

		if p.cur == before {
			p.next()
		}
	}
	return members, p.expectRbrace()
}

// parseInstanceDecl parses `instance [ TypeParams ] Class ( TypeArgs | "for"
// NamedType ) [ requires ... ] [ "{" { AssocTypeBind } "}" ]`.
func (p *parser) parseInstanceDecl(head ast.DeclHead) *ast.InstanceDecl {
	head.P = p.cur.Pos
	p.next() // 'instance'

	d := &ast.InstanceDecl{DeclHead: head}
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}

	d.Class = &ast.ClassRef{P: p.cur.Pos, N: p.expectIdent()}
	if p.at(lex.DOT) {
		p.next()
		d.Class.Qualifier, d.Class.N = d.Class.N, p.expectIdent()
	}
	d.N = d.Class.N

	switch {
	case p.at(lex.LT):
		d.Class.Args = p.parseTypeArgs()
	case p.accept(lex.FOR):
		d.For = p.parseCoreType()
	default:
		p.errs.add(p.cur.Pos, "expected type arguments or 'for', got %s", p.cur.Kind)
		p.syncTop()
		return d
	}

	if p.at(lex.REQUIRES) {
		d.Requires = p.parseClassRefs()
	}
	if p.at(lex.LBRACE) {
		d.Binds, d.End = p.parseAssocTypeBinds()
	}
	return d
}

func (p *parser) parseAssocTypeBinds() ([]*ast.AssocTypeBind, ast.Position) {
	p.next() // '{'

	var binds []*ast.AssocTypeBind
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur

		bind := &ast.AssocTypeBind{P: p.cur.Pos}
		if p.expect(lex.TYPE) {
			bind.N = p.expectIdent()
			if p.expect(lex.EQUAL) {
				bind.Target = p.parseTypeRef()
			}
			binds = append(binds, bind)
		}

		if p.cur == before {
			p.next()
		}
	}
	return binds, p.expectRbrace()
}
