// Package parser implements a hand-written recursive-descent parser that
// turns TDL source text into an [ast.File]. It collects every syntax
// error it finds in one pass rather than stopping at the first, so
// tooling like `tdl check` can report a complete list of problems.
package parser

import (
	"io"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/lex"
)

// Parse reads and parses a single TDL source file. On success it returns
// the parsed [ast.File]; on failure it returns a nil file and an
// *ErrorList describing every syntax error found.
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
	errs     ErrorList
}

func newParser(filename, src string) *parser {
	p := &parser{filename: filename, lx: lex.New(filename, src)}
	p.next()
	p.next()
	return p
}

func (p *parser) next() {
	p.cur = p.peek
	p.peek = p.lx.Next()
}

func (p *parser) expect(kind lex.Kind) {
	if p.cur.Kind != kind {
		p.errs.add(p.cur.Pos, "expected %s, got %s", kind, p.cur.Kind)
		return
	}
	p.next()
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

// syncTop skips tokens until one that can start (or end) a top-level
// declaration, so parseFile can recover from an unexpected token and keep
// collecting errors.
func (p *parser) syncTop() {
	for {
		switch p.cur.Kind {
		case lex.IMPORT, lex.TYPE, lex.ENUM, lex.AT, lex.EOF:
			return
		}
		p.next()
	}
}

// syncField skips tokens until the end of the enclosing type/enum body or
// the start of the next annotated field/value.
func (p *parser) syncField() {
	for {
		switch p.cur.Kind {
		case lex.RBRACE, lex.AT, lex.EOF:
			return
		}
		p.next()
	}
}

func (p *parser) parseFile() *ast.File {
	file := &ast.File{Filename: p.filename}

	if p.cur.Kind == lex.PACKAGE {
		file.Package = p.parsePackageDecl()
	}

	for p.cur.Kind != lex.EOF {
		switch p.cur.Kind {
		case lex.IMPORT:
			file.Imports = append(file.Imports, p.parseImportDecl())
		case lex.TYPE, lex.ENUM, lex.AT:
			ann := p.parseAnnotations()
			switch p.cur.Kind {
			case lex.TYPE:
				file.Types = append(file.Types, p.parseTypeDecl(ann))
			case lex.ENUM:
				file.Enums = append(file.Enums, p.parseEnumDecl(ann))
			default:
				p.errs.add(p.cur.Pos, "expected 'type' or 'enum' after annotations, got %s", p.cur.Kind)
				p.syncTop()
			}
		case lex.PACKAGE:
			p.errs.add(p.cur.Pos, "unexpected second 'package' declaration")
			p.parsePackageDecl()
		default:
			p.errs.add(p.cur.Pos, "unexpected token %s, expected a declaration", p.cur.Kind)
			p.next()
		}
	}

	return file
}

func (p *parser) parsePackageDecl() *ast.PackageDecl {
	pos := p.cur.Pos
	p.expect(lex.PACKAGE)
	name := p.parseDottedIdent()
	return &ast.PackageDecl{Pos: pos, Name: name}
}

func (p *parser) parseDottedIdent() string {
	parts := []string{p.expectIdent()}
	for p.cur.Kind == lex.DOT {
		p.next()
		parts = append(parts, p.expectIdent())
	}
	return strings.Join(parts, ".")
}

func (p *parser) parseImportDecl() *ast.ImportDecl {
	pos := p.cur.Pos
	p.expect(lex.IMPORT)

	if p.cur.Kind != lex.STRING {
		p.errs.add(p.cur.Pos, "expected string import path, got %s", p.cur.Kind)
		p.syncTop()
		return &ast.ImportDecl{Pos: pos}
	}
	path := p.cur.Text
	p.next()

	p.expect(lex.AS)
	alias := p.expectIdent()

	return &ast.ImportDecl{Pos: pos, Path: path, Alias: alias}
}

func (p *parser) parseAnnotations() []*ast.Annotation {
	var anns []*ast.Annotation
	for p.cur.Kind == lex.AT {
		anns = append(anns, p.parseAnnotation())
	}
	return anns
}

func (p *parser) parseAnnotation() *ast.Annotation {
	pos := p.cur.Pos
	p.expect(lex.AT)
	ns := p.expectIdent()

	var args []ast.AnnotationArg
	if p.cur.Kind == lex.LPAREN {
		p.next()
		if p.cur.Kind != lex.RPAREN {
			args = append(args, p.parseAnnotationArg())
			for p.cur.Kind == lex.COMMA {
				p.next()
				args = append(args, p.parseAnnotationArg())
			}
		}
		p.expect(lex.RPAREN)
	}

	return &ast.Annotation{Pos: pos, Namespace: ns, Args: args}
}

func (p *parser) parseAnnotationArg() ast.AnnotationArg {
	name := p.expectIdent()
	p.expect(lex.COLON)
	value := p.parseLiteral()
	return ast.AnnotationArg{Name: name, Value: value}
}

func (p *parser) parseTypeDecl(ann []*ast.Annotation) *ast.TypeDecl {
	pos := p.cur.Pos
	p.expect(lex.TYPE)
	name := p.expectIdent()
	p.expect(lex.LBRACE)

	var fields []*ast.Field
	for p.cur.Kind != lex.RBRACE && p.cur.Kind != lex.EOF {
		fieldAnn := p.parseAnnotations()
		if p.cur.Kind == lex.RBRACE {
			if len(fieldAnn) > 0 {
				p.errs.add(p.cur.Pos, "dangling annotation, expected a field")
			}
			break
		}
		if p.cur.Kind != lex.IDENT {
			p.errs.add(p.cur.Pos, "expected field name, got %s", p.cur.Kind)
			p.syncField()
			continue
		}
		fields = append(fields, p.parseField(fieldAnn))
	}
	p.expect(lex.RBRACE)

	return &ast.TypeDecl{Pos: pos, Name: name, Fields: fields, Annotations: ann}
}

func (p *parser) parseField(ann []*ast.Annotation) *ast.Field {
	pos := p.cur.Pos
	name := p.expectIdent()
	p.expect(lex.COLON)
	typ := p.parseTypeRef()

	optional := false
	if p.cur.Kind == lex.QUESTION {
		optional = true
		p.next()
	}

	var def *ast.Literal
	if p.cur.Kind == lex.EQUAL {
		p.next()
		def = p.parseLiteral()
	}

	return &ast.Field{Pos: pos, Name: name, Type: typ, Optional: optional, Default: def, Annotations: ann}
}

func (p *parser) parseEnumDecl(ann []*ast.Annotation) *ast.EnumDecl {
	pos := p.cur.Pos
	p.expect(lex.ENUM)
	name := p.expectIdent()
	p.expect(lex.LBRACE)

	var values []*ast.EnumValue
	for p.cur.Kind != lex.RBRACE && p.cur.Kind != lex.EOF {
		valAnn := p.parseAnnotations()
		if p.cur.Kind == lex.RBRACE {
			if len(valAnn) > 0 {
				p.errs.add(p.cur.Pos, "dangling annotation, expected an enum value")
			}
			break
		}
		if p.cur.Kind != lex.IDENT {
			p.errs.add(p.cur.Pos, "expected enum value name, got %s", p.cur.Kind)
			p.syncField()
			continue
		}
		values = append(values, p.parseEnumValue(valAnn))
	}
	p.expect(lex.RBRACE)

	return &ast.EnumDecl{Pos: pos, Name: name, Values: values, Annotations: ann}
}

func (p *parser) parseEnumValue(ann []*ast.Annotation) *ast.EnumValue {
	pos := p.cur.Pos
	name := p.expectIdent()

	var value *ast.Literal
	if p.cur.Kind == lex.EQUAL {
		p.next()
		value = p.parseLiteral()
	}

	return &ast.EnumValue{Pos: pos, Name: name, Value: value, Annotations: ann}
}

func (p *parser) parseTypeRef() ast.TypeRef {
	pos := p.cur.Pos
	if p.cur.Kind != lex.IDENT {
		p.errs.add(pos, "expected a type, got %s", p.cur.Kind)
		return ast.TypeRef{Pos: pos}
	}
	name := p.cur.Text
	p.next()

	switch name {
	case "list":
		p.expect(lex.LT)
		elem := p.parseTypeRef()
		p.expect(lex.GT)
		return ast.TypeRef{Pos: pos, List: &elem}
	case "map":
		p.expect(lex.LT)
		key := p.parseTypeRef()
		p.expect(lex.COMMA)
		val := p.parseTypeRef()
		p.expect(lex.GT)
		return ast.TypeRef{Pos: pos, MapKey: &key, MapValue: &val}
	}

	qualifier := ""
	if p.cur.Kind == lex.DOT {
		p.next()
		qualifier = name
		name = p.expectIdent()
	}
	return ast.TypeRef{Pos: pos, Qualifier: qualifier, Name: name}
}

func (p *parser) parseLiteral() *ast.Literal {
	pos := p.cur.Pos
	switch p.cur.Kind {
	case lex.STRING:
		v := p.cur.Text
		p.next()
		return &ast.Literal{Pos: pos, Kind: ast.LitString, Str: v}

	case lex.INT:
		text := p.cur.Text
		p.next()
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			p.errs.add(pos, "invalid integer literal %q: %v", text, err)
			return nil
		}
		return &ast.Literal{Pos: pos, Kind: ast.LitInt, Int: n}

	case lex.FLOAT:
		text := p.cur.Text
		p.next()
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			p.errs.add(pos, "invalid float literal %q: %v", text, err)
			return nil
		}
		return &ast.Literal{Pos: pos, Kind: ast.LitFloat, Float: f}

	case lex.TRUE:
		p.next()
		return &ast.Literal{Pos: pos, Kind: ast.LitBool, Bool: true}

	case lex.FALSE:
		p.next()
		return &ast.Literal{Pos: pos, Kind: ast.LitBool, Bool: false}

	case lex.LBRACK:
		p.next()
		var elems []*ast.Literal
		if p.cur.Kind != lex.RBRACK {
			elems = append(elems, p.parseLiteral())
			for p.cur.Kind == lex.COMMA {
				p.next()
				elems = append(elems, p.parseLiteral())
			}
		}
		p.expect(lex.RBRACK)
		return &ast.Literal{Pos: pos, Kind: ast.LitList, List: elems}

	default:
		p.errs.add(pos, "expected a literal value, got %s", p.cur.Kind)
		p.next()
		return nil
	}
}
