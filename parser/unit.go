package parser

import (
	"strconv"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

func (p *parser) parseUnitDecl(head ast.DeclHead) *ast.UnitDecl {
	head.P = p.cur.Pos
	p.next() // 'unit'

	d := &ast.UnitDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.accept(lex.EQUAL) {
		d.Expr = p.parseUnitExpr()
	}
	return d
}

// parseUnitExpr parses `UnitTerm { ( "*" | "/" ) UnitTerm }`.
//
// The expression is a flat sequence rather than a tree: `*` and `/` have
// equal precedence and associate left, so the terms carry their own
// operator and normalizing to base dimensions is the resolver's job.
func (p *parser) parseUnitExpr() *ast.UnitExpr {
	e := &ast.UnitExpr{P: p.cur.Pos}
	e.Terms = append(e.Terms, p.parseUnitTerm(""))

	for p.at(lex.STAR) || p.at(lex.SLASH) {
		op := p.cur.Text
		p.next()
		e.Terms = append(e.Terms, p.parseUnitTerm(op))
	}
	return e
}

func (p *parser) parseUnitTerm(op string) *ast.UnitTerm {
	t := &ast.UnitTerm{P: p.cur.Pos, Op: op, Exp: 1}

	if p.accept(lex.LPAREN) {
		t.Paren = p.parseUnitExpr()
		p.expect(lex.RPAREN)
	} else {
		t.N = p.expectIdent()
	}

	if p.accept(lex.CARET) {
		if !p.at(lex.INT) {
			p.errs.add(p.cur.Pos, "expected a unit exponent, got %s", p.cur.Kind)
			return t
		}
		exp, err := strconv.Atoi(p.cur.Text)
		if err != nil {
			p.errs.add(p.cur.Pos, "unit exponent out of range: %s", p.cur.Text)
		}
		t.Exp = exp
		p.next()
	}
	return t
}

// parseTypeArgs parses a `<...>` argument list. Each argument is a type or
// a unit, and only operators tell them apart: a bare name could be either,
// so it is recorded as a type reference and the resolver decides by kind.
func (p *parser) parseTypeArgs() []*ast.TypeArg {
	p.next() // '<'

	var args []*ast.TypeArg
	for {
		args = append(args, p.parseTypeArg())
		if !p.accept(lex.COMMA) {
			break
		}
	}
	p.expect(lex.GT)
	return args
}

func (p *parser) parseTypeArg() *ast.TypeArg {
	arg := &ast.TypeArg{P: p.cur.Pos}

	// A parenthesized argument can only be a unit expression: no type
	// reference form starts with '('.
	if p.at(lex.LPAREN) {
		arg.Unit = p.parseUnitExpr()
		return arg
	}

	ref := p.parseTypeRef()

	// An operator makes it unambiguously a unit. A named reference with no
	// arguments and no sugar is the only thing that can become one.
	if (p.at(lex.STAR) || p.at(lex.SLASH) || p.at(lex.CARET)) && plainName(ref) {
		arg.Unit = p.continueUnitExpr(ref)
		return arg
	}

	arg.Type = ref
	return arg
}

// plainName reports whether ref is a bare unqualified name, the only shape
// a unit can have been written as.
func plainName(ref *ast.TypeRef) bool {
	return ref.N != "" && ref.Qualifier == "" && len(ref.Args) == 0 &&
		!ref.Optional && !ref.Nullable
}

// continueUnitExpr rebuilds a unit expression whose first term was already
// consumed as a type reference.
func (p *parser) continueUnitExpr(first *ast.TypeRef) *ast.UnitExpr {
	term := &ast.UnitTerm{P: first.P, N: first.N, Exp: 1}
	if p.accept(lex.CARET) {
		if p.at(lex.INT) {
			exp, err := strconv.Atoi(p.cur.Text)
			if err != nil {
				p.errs.add(p.cur.Pos, "unit exponent out of range: %s", p.cur.Text)
			}
			term.Exp = exp
			p.next()
		} else {
			p.errs.add(p.cur.Pos, "expected a unit exponent, got %s", p.cur.Kind)
		}
	}

	e := &ast.UnitExpr{P: first.P, Terms: []*ast.UnitTerm{term}}
	for p.at(lex.STAR) || p.at(lex.SLASH) {
		op := p.cur.Text
		p.next()
		e.Terms = append(e.Terms, p.parseUnitTerm(op))
	}
	return e
}
