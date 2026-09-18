package lsp

import (
	"errors"
	"strings"
	"sync"

	"go.lsp.dev/uri"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// document is one file the editor has open.
//
// The text is the editor's, not the disk's, which is the whole point: a
// file being edited has not been saved, and resolving an import against
// what is on disk would answer a question nobody asked.
type document struct {
	uri     uri.URI
	path    string
	version int32
	text    string

	snap *snapshot // nil when the text has changed since it was computed
}

// snapshot is everything derived from one version of a document's text.
//
// It is computed on demand and thrown away whole when the text changes.
// Nothing here is incremental, because a .tdl file is a description of a
// domain model rather than a program and reparsing one costs nothing worth
// measuring.
type snapshot struct {
	index *lineIndex
	file  *ast.File
	errs  parser.ErrorList
	model *ir.Model
	diags sema.Diagnostics
	refs  sema.References
}

// store holds every open document.
//
// It is the server's whole mutable state, so it owns the lock rather than
// the server: every request that reads a document goes through here, and a
// notification that replaces one does too.
type store struct {
	mu   sync.Mutex
	docs map[string]*document // keyed by path, which is what sema works in
}

func newStore() *store {
	return &store{docs: map[string]*document{}}
}

// open records a document the editor has opened, replacing any copy of it.
func (s *store) open(u uri.URI, version int32, text string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := &document{uri: u, path: u.FsPath(), version: version, text: text}
	s.docs[doc.path] = doc
	return doc
}

// change replaces a document's text. It reports the document, or nil when
// the editor sent a change for a file it never opened.
func (s *store) change(u uri.URI, version int32, text string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.docs[u.FsPath()]
	if !ok {
		return nil
	}
	doc.version, doc.text, doc.snap = version, text, nil
	return doc
}

// close forgets a document, so an import of it reads the disk again.
func (s *store) close(u uri.URI) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, u.FsPath())
}

// invalidate drops every snapshot.
//
// A snapshot depends on the imports it read as well as on its own text, so
// a change to any open document, and a close that hands one back to the
// disk, makes every other one stale. There are as many open documents as a
// person has tabs, and reanalyzing one is a parse and a lowering, so
// tracking which snapshot read which file would buy nothing yet.
func (s *store) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, doc := range s.docs {
		doc.snap = nil
	}
}

// get returns the document at a path, or nil.
func (s *store) get(path string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docs[path]
}

// paths returns every open document's path.
func (s *store) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, 0, len(s.docs))
	for path := range s.docs {
		out = append(out, path)
	}
	return out
}

// analyze computes a document's snapshot, reusing the one it has when the
// text has not changed.
//
// Lowering runs only when the file parses. A tree with holes in it reports
// names it cannot resolve because the declaration naming them failed to
// parse, and those are noise: sema.Diagnostics already states the same
// rule for its own passes, that a non-empty list means no later pass
// should run.
func (s *store) analyze(doc *document, opts ...sema.Option) *snapshot {
	s.mu.Lock()
	if doc.snap != nil {
		defer s.mu.Unlock()
		return doc.snap
	}
	text := doc.text
	path := doc.path
	s.mu.Unlock()

	snap := &snapshot{index: newLineIndex(text)}
	file, err := parser.Parse(path, strings.NewReader(text))
	snap.file = file

	var errs parser.ErrorList
	switch {
	case err == nil:
		// The options are copied rather than appended to: the slice
		// belongs to the server and is shared by every analysis, and
		// append would be free to write into it.
		with := make([]sema.Option, 0, len(opts)+1)
		with = append(append(with, opts...), sema.WithReferences(&snap.refs))
		snap.model, snap.diags = sema.Lower(file, with...)
	case errors.As(err, &errs):
		snap.errs = errs
	default:
		// Parse returns an ErrorList or nothing, so this is a read error
		// that cannot happen over a string. Report it at the top of the
		// file rather than dropping it.
		snap.errs = parser.ErrorList{{Msg: err.Error()}}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if doc.text == text {
		doc.snap = snap
	}
	return snap
}
