package parser

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// parseTargetDecl parses `target go for billing { ... }`.
func (p *parser) parseTargetDecl(head ast.DeclHead) *ast.TargetDecl {
	head.P = p.cur.Pos
	p.next() // 'target'

	d := &ast.TargetDecl{DeclHead: head}
	d.N = p.expectIdent()
	if !p.expect(lex.FOR) {
		p.syncTop()
		return d
	}
	d.For = p.parseDottedIdent()
	d.Entries = p.parseTargetEntries()
	return d
}

func (p *parser) parseTargetEntries() []*ast.TargetEntry {
	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return nil
	}

	var entries []*ast.TargetEntry
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		entries = append(entries, p.parseTargetEntry())
		if p.cur == before {
			p.next()
		}
	}
	p.expect(lex.RBRACE)
	return entries
}

// parseTargetEntry parses one of the three entry forms:
//
//	Path { ... }        a nested block scoping a path
//	Path => Directive   a directive applied to that path
//	Directive           a directive applying to the enclosing scope
//
// All three start with an identifier, so the token after the path decides.
func (p *parser) parseTargetEntry() *ast.TargetEntry {
	entry := &ast.TargetEntry{P: p.cur.Pos}

	pos := p.cur.Pos
	name := p.expectDirectiveIdent()
	dotted := name
	for p.at(lex.DOT) {
		p.next()
		dotted += "." + p.expectDirectiveIdent()
	}

	switch {
	case p.at(lex.LBRACE):
		entry.Path = dotted
		entry.Entries = p.parseTargetEntries()
	case p.accept(lex.FATARROW):
		entry.Path = dotted
		entry.Directive = p.parseDirective()
	default:
		if dotted != name {
			p.errs.add(pos, "a bare directive is a single name, got the path %s", dotted)
		}
		entry.Directive = p.finishDirective(pos, name)
	}
	return entry
}

func (p *parser) parseDirective() *ast.Directive {
	pos := p.cur.Pos
	return p.finishDirective(pos, p.expectDirectiveIdent())
}

// expectDirectiveIdent reads a name inside a target block, accepting
// reserved keywords. Directives are opaque and their namespace belongs to
// the backend, so `package("github.com/acme/billing")` is a directive named
// `package` rather than a syntax error. Model paths cannot collide with
// this: a declaration name is always an ordinary identifier.
func (p *parser) expectDirectiveIdent() string {
	if p.cur.Kind != lex.IDENT && !lex.IsKeyword(p.cur.Text) {
		p.errs.add(p.cur.Pos, "expected a directive or path name, got %s", p.cur.Kind)
		return ""
	}
	name := p.cur.Text
	p.next()
	return name
}

// finishDirective parses a directive's argument list. Arguments are
// parenthesized: whitespace is insignificant, so an unparenthesized list
// could not be told from the entry that follows it.
func (p *parser) finishDirective(pos lex.Position, name string) *ast.Directive {
	d := &ast.Directive{P: pos, N: name}
	if !p.accept(lex.LPAREN) {
		return d
	}
	for !p.at(lex.RPAREN) && !p.at(lex.EOF) {
		d.Args = append(d.Args, p.parseLiteral())
		if !p.accept(lex.COMMA) {
			break
		}
	}
	p.expect(lex.RPAREN)
	return d
}
