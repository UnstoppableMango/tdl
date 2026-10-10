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
	d.For = p.parsePackagePath()
	d.Entries, d.Rbrace = p.parseTargetEntries()
	return d
}

func (p *parser) parseTargetEntries() ([]*ast.TargetEntry, ast.Position) {
	if !p.expect(lex.LBRACE) {
		p.syncTop()
		return nil, ast.Position{}
	}

	var entries []*ast.TargetEntry
	p.untilRbrace(func() { entries = append(entries, p.parseTargetEntry()) })
	return entries, p.expectRbrace()
}

// parseTargetEntry parses one of the three entry forms:
//
//	Path { ... }        a nested block scoping a path
//	Path => Directive   a directive applied to that path
//	Directive           a directive applying to the enclosing scope
func (p *parser) parseTargetEntry() *ast.TargetEntry {
	entry := &ast.TargetEntry{P: p.cur.Pos}
	defer func() { entry.E = p.endOf(entry.P) }()

	pos := p.cur.Pos
	name := p.expectName("directive or path name")
	dotted := name
	for p.at(lex.DOT) {
		p.next()
		dotted += "." + p.expectName("directive or path name")
	}

	switch {
	case p.at(lex.LBRACE):
		entry.Path = dotted
		entry.Entries, entry.Rbrace = p.parseTargetEntries()
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
	return p.finishDirective(pos, p.expectName("directive or path name"))
}

// expectName reads a name that may be a reserved keyword, as directive
// names, target paths, and package path segments may be; kind names it in
// the error.
func (p *parser) expectName(kind string) string {
	if p.cur.Kind != lex.IDENT && !lex.IsKeyword(p.cur.Text) {
		p.errs.add(p.cur.Pos, "expected a %s, got %s", kind, p.cur.Kind)
		return ""
	}
	name := p.cur.Text
	p.next()
	return name
}

// finishDirective parses a directive's optional argument list.
func (p *parser) finishDirective(pos lex.Position, name string) *ast.Directive {
	d := &ast.Directive{P: pos, N: name}
	defer func() { d.E = p.endOf(pos) }()
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
