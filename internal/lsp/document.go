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

// document is one file the editor has open, holding the editor's unsaved
// text.
type document struct {
	uri     uri.URI
	path    string
	version int32
	text    string

	snap *snapshot // nil when the text has changed since it was computed
}

// snapshot is everything derived from one version of a document's text,
// computed on demand and discarded whole when the text changes.
type snapshot struct {
	index *lineIndex
	file  *ast.File
	errs  parser.ErrorList
	model *ir.Model
	diags sema.Diagnostics
	refs  sema.References
}

// store holds every open document. It is the server's only mutable state
// and owns the lock.
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

// invalidate drops every snapshot. A snapshot depends on the imports it
// read, so a change to or close of any document makes the others stale;
// tracking which snapshot read which file is not worth it at this scale.
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
// text has not changed. Lowering runs only when the file parses, since a
// tree with holes reports spurious undefined names.
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
		// Copy: opts is shared by every analysis, and append could write
		// into it.
		with := make([]sema.Option, 0, len(opts)+1)
		with = append(append(with, opts...), sema.WithReferences(&snap.refs))
		snap.model, snap.diags = sema.Lower(file, with...)
	case errors.As(err, &errs):
		snap.errs = errs
	default:
		// A read error, which a string reader does not produce.
		snap.errs = parser.ErrorList{{Msg: err.Error()}}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if doc.text == text {
		doc.snap = snap
	}
	return snap
}
