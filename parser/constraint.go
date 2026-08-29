package parser

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// parseConstraintBlock parses `where "{" { Constraint } "}"`.
//
// The `where` prefix is what keeps `{` unambiguous: after a complete type
// reference it could otherwise open a set type, a declaration body, or this.
func (p *parser) parseConstraintBlock() []*ast.Constraint {
	p.next() // 'where'

	if !p.expect(lex.LBRACE) {
		return nil
	}

	var constraints []*ast.Constraint
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		constraints = append(constraints, p.parseConstraint())
		if p.cur == before {
			p.next()
		}
	}
	p.expect(lex.RBRACE)
	return constraints
}

// parseConstraint parses `identifier [ "(" [ Arg { "," Arg } ] ")" ]`.
//
// Constraint names are ordinary identifiers, not keywords: the set is open,
// so the parser recognizes no name in particular. Arguments are
// parenthesized for the same reason a directive's are, since without a
// delimiter `min 0 max 100` could not be split.
func (p *parser) parseConstraint() *ast.Constraint {
	c := &ast.Constraint{P: p.cur.Pos, N: p.expectIdent()}

	if !p.accept(lex.LPAREN) {
		return c
	}
	for !p.at(lex.RPAREN) && !p.at(lex.EOF) {
		c.Args = append(c.Args, p.parseConstraintArg())
		if !p.accept(lex.COMMA) {
			break
		}
	}
	p.expect(lex.RPAREN)
	return c
}

// parseConstraintArg parses a range, a regex, or an ordinary literal.
func (p *parser) parseConstraintArg() *ast.Literal {
	pos := p.cur.Pos

	// `..254`: a range open below.
	if p.at(lex.RANGE) {
		p.next()
		return &ast.Literal{P: pos, Kind: ast.LitRange, Hi: p.parseRangeBound()}
	}

	// A regex is scanned on request: `/` is also division in a unit
	// expression, and the token before it is an ordinary identifier either
	// way.
	if p.at(lex.SLASH) {
		tok := p.rescanRegexAtCurrent()
		p.next()
		return &ast.Literal{P: tok.Pos, Kind: ast.LitRegex, Text: tok.Text}
	}

	lit := p.parseLiteral()

	// `3..254` and `1..`: a range that started with its lower bound.
	if lit.Kind == ast.LitInt && p.at(lex.RANGE) {
		p.next()
		return &ast.Literal{P: pos, Kind: ast.LitRange, Lo: lit, Hi: p.parseRangeBound()}
	}
	return lit
}

// parseRangeBound reads the upper bound of a range, which may be absent.
func (p *parser) parseRangeBound() *ast.Literal {
	if !p.at(lex.INT) {
		return nil
	}
	lit := &ast.Literal{P: p.cur.Pos, Kind: ast.LitInt, Text: p.cur.Text}
	p.next()
	return lit
}

// rescanRegexAtCurrent rescans the input from the current token as a regex
// literal and re-primes the lookahead behind it.
func (p *parser) rescanRegexAtCurrent() lex.Token {
	tok := p.lx.RescanRegexAt(p.cur.Pos)
	if tok.Kind == lex.ILLEGAL {
		p.errs.add(tok.Pos, "unterminated regex literal")
	}
	p.cur = tok
	p.peek = p.lx.Next()
	return tok
}
