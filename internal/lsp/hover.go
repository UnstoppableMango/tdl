package lsp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/prelude"
)

// Hover describes the name under the cursor: the declaration it refers to
// in canonical form, its deprecation, and its doc comment. A reference is
// answered from the index Definition reads, including one into the
// prelude; a declaration's own name is found in the tree.
func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc := s.store.get(params.TextDocument.URI.FsPath())
	if doc == nil {
		return nil, nil
	}

	snap := s.store.analyze(doc, s.opts...)
	if ref, ok := snap.refs.At(doc.path, snap.index.byteOffset(params.Position)); ok {
		if ref.Target.Filename == "" {
			return nil, nil // unresolved, or qualified into a dependency
		}
		rng := snap.index.span(ref.Pos.Offset, ref.Len)
		if decl := s.declAt(ref.Target); decl != nil {
			return hover(describe(decl), rng), nil
		}
		// A type parameter starts no declaration and has no ref.Decl.
		if ref.Decl == nil {
			return hover("```tdl\n"+ref.Name+"\n```\n\nType parameter.", rng), nil
		}
		return nil, nil
	}

	if snap.file == nil {
		return nil, nil
	}
	for _, decl := range snap.file.Decls {
		if _, ok := decl.(*ast.TargetDecl); ok {
			continue // names a backend, which declares nothing here
		}
		rng, ok := snap.index.nameSpan(decl.Pos(), decl.Head().N)
		if ok && within(rng, params.Position) {
			return hover(describe(decl), rng), nil
		}
	}
	return nil, nil
}

// describe renders a declaration as markdown: its canonical form, then its
// deprecation, then its doc comment.
func describe(decl ast.Decl) string {
	var b strings.Builder
	b.WriteString("```tdl\n")
	b.WriteString(ast.PrintDecl(decl))
	b.WriteString("```\n")

	head := decl.Head()
	if head.Dep != nil {
		b.WriteString("\n**Deprecated**")
		if head.Dep.Reason != "" {
			b.WriteString(": " + head.Dep.Reason)
		}
		b.WriteString("\n")
	}
	if len(head.Doc) > 0 {
		b.WriteString("\n" + strings.Join(head.Doc, "\n") + "\n")
	}
	return b.String()
}

func hover(markdown string, rng protocol.Range) *protocol.Hover {
	return &protocol.Hover{
		Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: markdown},
		Range:    &rng,
	}
}

// declAt is the declaration that starts at pos.
func (s *Server) declAt(pos ast.Position) ast.Decl {
	file := s.parsed(pos.Filename)
	if file == nil {
		return nil
	}
	for _, decl := range file.Decls {
		if p := decl.Pos(); p.Line == pos.Line && p.Col == pos.Col {
			return decl
		}
	}
	return nil
}

// parsed is the tree of the file a reference points into: the editor's
// text when open, the disk's when not, and the embedded source for the
// prelude. A file that fails to parse yields what the parser recovered.
func (s *Server) parsed(path string) *ast.File {
	if path == prelude.Name {
		return preludeFile()
	}
	if doc := s.store.get(path); doc != nil {
		return s.store.analyze(doc, s.opts...).file
	}
	if !filepath.IsAbs(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	file, _ := parser.Parse(path, strings.NewReader(string(data)))
	return file
}

// preludeFile is the embedded prelude's tree, parsed once.
var preludeFile = sync.OnceValue(func() *ast.File {
	file, _ := parser.Parse(prelude.Name, strings.NewReader(prelude.Source))
	return file
})

// within reports whether a position falls inside a range, end included,
// so a cursor just past a name is still on it.
func within(rng protocol.Range, pos protocol.Position) bool {
	before := func(a, b protocol.Position) bool {
		return a.Line < b.Line || a.Line == b.Line && a.Character <= b.Character
	}
	return before(rng.Start, pos) && before(pos, rng.End)
}
