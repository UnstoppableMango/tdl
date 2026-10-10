// Package parser is a recursive-descent parser turning TDL source into an
// [ast.File]. It reports every syntax error in one pass.
package parser

import (
	"io"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// Parse parses one TDL source file. On failure it returns a nil file and
// an [ErrorList].
func Parse(filename string, r io.Reader) (*ast.File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p := newParser(filename, string(data))
	file := p.parseFile()
	if len(p.errs) > 0 {
		return nil, p.errs
	}
	return file, nil
}

type parser struct {
	filename string
	lx       *lex.Lexer
	cur      lex.Token
	peek     lex.Token
	prev     ast.Position // just past the last token consumed
	errs     ErrorList
}

func newParser(filename, src string) *parser {
	p := &parser{filename: filename, lx: lex.New(filename, src)}
	p.next()
	p.next()
	return p
}

func (p *parser) next() {
	p.prev = p.cur.End
	p.cur = p.peek
	p.peek = p.lx.Next()
}

// endOf is where a node that started at start ends: just past the last
// token consumed, or start itself when the node consumed none.
func (p *parser) endOf(start ast.Position) ast.Position {
	if p.prev.Line == 0 || p.prev.Offset < start.Offset {
		return start
	}
	return p.prev
}

// endDecl records where a declaration ends and returns it.
func (p *parser) endDecl(d ast.Decl) ast.Decl {
	h := d.Head()
	h.E = p.endOf(h.P)
	return d
}

func (p *parser) at(kind lex.Kind) bool { return p.cur.Kind == kind }

func (p *parser) accept(kind lex.Kind) bool {
	if p.cur.Kind == kind {
		p.next()
		return true
	}
	return false
}

func (p *parser) expect(kind lex.Kind) bool {
	if p.cur.Kind != kind {
		p.errs.add(p.cur.Pos, "expected %s, got %s", kind, p.cur.Kind)
		return false
	}
	p.next()
	return true
}

// expectRbrace consumes a block's closing brace and returns its position,
// which the formatter needs to place a comment on a block's last line. On
// a missing brace it returns the position of the token found instead.
func (p *parser) expectRbrace() ast.Position {
	pos := p.cur.Pos
	p.expect(lex.RBRACE)
	return pos
}

// untilRbrace runs fn until a closing brace or EOF, dropping a token
// whenever fn made no progress so a bad item cannot loop forever.
func (p *parser) untilRbrace(fn func()) {
	for !p.at(lex.RBRACE) && !p.at(lex.EOF) {
		before := p.cur
		fn()
		if p.cur == before {
			p.next()
		}
	}
}

// parseQualified reads `name` or `alias.name`.
func (p *parser) parseQualified() (qualifier, name string) {
	name = p.expectIdent()
	if p.at(lex.DOT) {
		p.next()
		qualifier, name = name, p.expectIdent()
	}
	return qualifier, name
}

func (p *parser) expectIdent() string {
	if p.cur.Kind != lex.IDENT {
		p.errs.add(p.cur.Pos, "expected identifier, got %s", p.cur.Kind)
		return ""
	}
	name := p.cur.Text
	p.next()
	return name
}

// declStart reports whether kind can begin a top-level declaration.
func declStart(kind lex.Kind) bool {
	switch kind {
	case lex.IMPORT, lex.PRIMITIVE, lex.UNIT, lex.ALIAS, lex.TYPE, lex.ENUM,
		lex.CLASS, lex.MIXIN, lex.INSTANCE, lex.TARGET, lex.DOC, lex.EOF:
		return true
	}
	return false
}

// syncTop skips to the next token that can start a top-level declaration.
func (p *parser) syncTop() {
	for !declStart(p.cur.Kind) {
		p.next()
	}
}

func (p *parser) parseFile() *ast.File {
	file := &ast.File{Filename: p.filename}

	for !p.at(lex.EOF) {
		doc, docP := p.parseDoc()
		head := ast.DeclHead{Doc: doc, DocP: docP}
		if p.atContextual("deprecated") {
			head.Dep = p.parseDeprecated()
		}

		switch p.cur.Kind {
		case lex.IMPORT:
			file.Imports = append(file.Imports, p.parseImportDecl(head))
		case lex.PRIMITIVE:
			file.Decls = append(file.Decls, p.endDecl(p.parsePrimitiveDecl(head)))
		case lex.ALIAS:
			file.Decls = append(file.Decls, p.endDecl(p.parseAliasDecl(head)))
		case lex.UNIT:
			file.Decls = append(file.Decls, p.endDecl(p.parseUnitDecl(head)))
		case lex.TYPE:
			file.Decls = append(file.Decls, p.endDecl(p.parseTypeDecl(head)))
		case lex.MIXIN:
			file.Decls = append(file.Decls, p.endDecl(p.parseStructDecl(head)))
		case lex.ENUM:
			file.Decls = append(file.Decls, p.endDecl(p.parseEnumDecl(head)))
		case lex.CLASS:
			file.Decls = append(file.Decls, p.endDecl(p.parseClassDecl(head)))
		case lex.INSTANCE:
			file.Decls = append(file.Decls, p.endDecl(p.parseInstanceDecl(head)))
		case lex.TARGET:
			file.Decls = append(file.Decls, p.endDecl(p.parseTargetDecl(head)))
		case lex.PACKAGE:
			pkg := p.parsePackageDecl(head)
			switch {
			case file.Package != nil:
				p.errs.add(pkg.P, "unexpected second 'package' declaration")
			case head.Dep != nil:
				p.errs.add(pkg.P, "a 'package' declaration cannot be deprecated")
			case len(file.Imports) > 0 || len(file.Decls) > 0:
				p.errs.add(pkg.P, "'package' must come before imports and declarations")
			default:
				file.Package = pkg
			}
		case lex.EOF:
			p.errs.add(p.cur.Pos, "doc comment at end of file, attached to nothing")
		default:
			p.errs.add(p.cur.Pos, "unexpected token %s, expected a declaration", p.cur.Kind)
			p.next()
			p.syncTop()
		}
	}

	// At EOF the lexer has collected every comment.
	file.EOF = p.cur.Pos
	for _, c := range p.lx.Comments() {
		file.Comments = append(file.Comments, &ast.Comment{P: c.Pos, Text: c.Text})
	}

	return file
}

// parseDoc consumes a run of `///` lines and returns where each was written,
// which the formatter uses to order them against ordinary comments.
func (p *parser) parseDoc() ([]string, []ast.Position) {
	var doc []string
	var pos []ast.Position
	for p.at(lex.DOC) {
		doc = append(doc, p.cur.Text)
		pos = append(pos, p.cur.Pos)
		p.next()
	}
	return doc, pos
}

func (p *parser) parsePackageDecl(head ast.DeclHead) *ast.PackageDecl {
	pos := p.cur.Pos
	p.next() // 'package'
	d := &ast.PackageDecl{Doc: head.Doc, DocP: head.DocP, P: pos, Path: p.parsePackagePath()}
	d.E = p.endOf(pos)
	return d
}

// parsePackagePath parses `Name { . Name }` after `package` or `for`.
func (p *parser) parsePackagePath() string {
	path := p.expectName("package name")
	for p.at(lex.DOT) {
		p.next()
		path += "." + p.expectName("package name")
	}
	return path
}

func (p *parser) parseDottedIdent() string {
	name := p.expectIdent()
	for p.at(lex.DOT) {
		p.next()
		name += "." + p.expectIdent()
	}
	return name
}

func (p *parser) parseImportDecl(head ast.DeclHead) *ast.ImportDecl {
	pos := p.cur.Pos
	p.next() // 'import'

	imp := &ast.ImportDecl{Doc: head.Doc, DocP: head.DocP, P: pos}
	defer func() { imp.E = p.endOf(pos) }()
	if p.at(lex.STRING) {
		imp.Path = p.cur.Text
		p.next()
	} else {
		p.errs.add(p.cur.Pos, "expected import path string, got %s", p.cur.Kind)
		p.syncTop()
		return imp
	}

	if !p.expect(lex.AS) {
		p.syncTop()
		return imp
	}
	imp.Alias = p.expectIdent()
	return imp
}

func (p *parser) parsePrimitiveDecl(head ast.DeclHead) *ast.PrimitiveDecl {
	head.P = p.cur.Pos
	p.next() // 'primitive'

	d := &ast.PrimitiveDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.accept(lex.COLON) {
		d.Kind = p.parseKind()
	}
	return d
}

func (p *parser) parseAliasDecl(head ast.DeclHead) *ast.AliasDecl {
	head.P = p.cur.Pos
	p.next() // 'alias'

	d := &ast.AliasDecl{DeclHead: head}
	d.N = p.expectIdent()
	if p.at(lex.LT) {
		d.Params = p.parseTypeParams()
	}
	if !p.expect(lex.EQUAL) {
		p.syncTop()
		return d
	}
	d.Target = p.parseTypeRef()
	return d
}

func (p *parser) parseTypeParams() []*ast.TypeParam {
	p.next() // '<'

	var params []*ast.TypeParam
	for {
		param := &ast.TypeParam{P: p.cur.Pos}
		param.N = p.expectIdent()
		if p.accept(lex.COLON) {
			param.Kind = p.parseKind()
		}
		param.E = p.endOf(param.P)
		params = append(params, param)

		if !p.accept(lex.COMMA) {
			break
		}
	}
	p.expect(lex.GT)
	return params
}

// parseKind parses `Kind = KindAtom { "->" KindAtom }`, right-associative.
func (p *parser) parseKind() *ast.Kind {
	k := &ast.Kind{P: p.cur.Pos}
	defer func() { k.E = p.endOf(k.P) }()

	switch {
	case p.at(lex.TYPE), p.at(lex.UNIT):
		k.N = p.cur.Text
		p.next()
	case p.accept(lex.LPAREN):
		k.Paren = p.parseKind()
		p.expect(lex.RPAREN)
	default:
		p.errs.add(p.cur.Pos, "expected a kind, got %s", p.cur.Kind)
		p.next()
		return k
	}

	if p.accept(lex.ARROW) {
		k.Arrow = p.parseKind()
	}
	return k
}

// parseTypeRef parses `TypeRef = CoreType [ "?" ] [ "|" "null" ]`.
func (p *parser) parseTypeRef() *ast.TypeRef {
	t := p.parseCoreType()
	defer func() { t.E = p.endOf(t.P) }()

	if p.accept(lex.QUESTION) {
		t.Optional = true
	}
	if p.at(lex.PIPE) {
		p.next()
		if !p.expect(lex.NULL) {
			return t
		}
		t.Nullable = true
	}
	return t
}

// parseCoreType parses the list, set, map, and named forms. Bracket sugar
// is recorded as written; lowering resolves it.
func (p *parser) parseCoreType() *ast.TypeRef {
	pos := p.cur.Pos

	switch {
	case p.accept(lex.LBRACK):
		t := &ast.TypeRef{P: pos, List: p.parseTypeRef()}
		p.expect(lex.RBRACK)
		t.E = p.endOf(pos)
		return t

	case p.accept(lex.LBRACE):
		inner := p.parseTypeRef()
		t := &ast.TypeRef{P: pos}
		if p.accept(lex.ARROW) {
			t.MapKey, t.MapValue = inner, p.parseTypeRef()
		} else {
			t.Set = inner
		}
		p.expect(lex.RBRACE)
		t.E = p.endOf(pos)
		return t
	}

	t := &ast.TypeRef{P: pos}
	t.Qualifier, t.N = p.parseQualified()
	if p.at(lex.LT) {
		t.Args = p.parseTypeArgs()
	}
	t.E = p.endOf(pos)
	return t
}
