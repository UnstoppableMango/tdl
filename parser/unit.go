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

// parseUnitExpr parses `UnitTerm { ( "*" | "/" ) UnitTerm }` as a flat
// list of terms, each carrying its operator.
func (p *parser) parseUnitExpr() *ast.UnitExpr {
	e := &ast.UnitExpr{P: p.cur.Pos}
	e.Terms = append(e.Terms, p.parseUnitTerm(""))
	return p.parseUnitTerms(e)
}

// parseUnitTerms reads the `( "*" | "/" ) UnitTerm` pairs after the first.
func (p *parser) parseUnitTerms(e *ast.UnitExpr) *ast.UnitExpr {
	for p.at(lex.STAR) || p.at(lex.SLASH) {
		op := p.cur.Text
		p.next()
		e.Terms = append(e.Terms, p.parseUnitTerm(op))
	}
	e.E = p.endOf(e.P)
	return e
}

func (p *parser) parseUnitTerm(op string) *ast.UnitTerm {
	t := &ast.UnitTerm{P: p.cur.Pos, Op: op}

	if p.accept(lex.LPAREN) {
		t.Paren = p.parseUnitExpr()
		p.expect(lex.RPAREN)
	} else {
		t.N = p.expectIdent()
	}
	t.Exp = p.parseExponent()
	t.E = p.endOf(t.P)
	return t
}

// parseExponent reads an optional `^ int`, defaulting to 1.
func (p *parser) parseExponent() int {
	if !p.accept(lex.CARET) {
		return 1
	}
	if !p.at(lex.INT) {
		p.errs.add(p.cur.Pos, "expected a unit exponent, got %s", p.cur.Kind)
		return 1
	}
	exp, err := strconv.Atoi(p.cur.Text)
	if err != nil {
		p.errs.add(p.cur.Pos, "unit exponent out of range: %s", p.cur.Text)
	}
	p.next()
	return exp
}

// parseTypeArgs parses a `<...>` list of types and units. A bare name is
// recorded as a type reference; lowering decides by kind.
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
	defer func() { arg.E = p.endOf(arg.P) }()

	// No type reference starts with '(', so this is a unit.
	if p.at(lex.LPAREN) {
		arg.Unit = p.parseUnitExpr()
		return arg
	}

	ref := p.parseTypeRef()

	// An operator after a plain name makes it a unit.
	if (p.at(lex.STAR) || p.at(lex.SLASH) || p.at(lex.CARET)) && plainName(ref) {
		arg.Unit = p.continueUnitExpr(ref)
		return arg
	}

	arg.Type = ref
	return arg
}

// plainName reports whether ref is a bare unqualified name.
func plainName(ref *ast.TypeRef) bool {
	return ref.N != "" && ref.Qualifier == "" && len(ref.Args) == 0 &&
		!ref.Optional && !ref.Nullable
}

// continueUnitExpr rebuilds a unit expression whose first term was already
// consumed as a type reference.
func (p *parser) continueUnitExpr(first *ast.TypeRef) *ast.UnitExpr {
	term := &ast.UnitTerm{P: first.P, N: first.N, Exp: p.parseExponent()}
	term.E = p.endOf(term.P)
	return p.parseUnitTerms(&ast.UnitExpr{P: first.P, Terms: []*ast.UnitTerm{term}})
}
