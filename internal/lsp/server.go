// Package lsp serves the Language Server Protocol over a TDL model being
// edited: what is wrong with a file, where a name under the cursor was
// declared, what it is, the file's outline, and its canonical form.
//
// It is private for the reason internal/sema is. `ir` and `proto` are the
// compatibility surface, and an editor integration is not.
//
// See docs/design/lsp.md for the reasoning and docs/design/lsp-plan.md for
// what each phase adds.
package lsp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/unstoppablemango/tdl/internal/sema"
)

// source is what a diagnostic says produced it, shown by editors beside
// the message.
const source = "tdl"

// errNoClient answers a request whose context carries no client. Serve
// puts one there, so reaching this means the server was driven by
// something that did not.
var errNoClient = errors.New("lsp: no client in context")

// Server answers protocol requests about the documents an editor has open.
//
// It embeds protocol.UnimplementedServer, so a request it does not serve
// is answered with "method not found" rather than with a panic, and adding
// a feature is writing the method for it.
type Server struct {
	protocol.UnimplementedServer

	store *store
	opts  []sema.Option
}

// NewServer returns a server.
//
// The client it publishes to comes from each request's context rather than
// from a field, which is how protocol.NewServer hands one over: the
// connection exists before the client dispatching across it, and the
// server exists before the connection.
//
// opts are the lowering options every document is analyzed with, which is
// how a caller replaces the prelude. The loader is not among them: the
// server supplies its own, so an open document's unsaved text is what an
// import of it resolves to.
func NewServer(opts ...sema.Option) *Server {
	s := &Server{store: newStore()}
	s.opts = append(append([]sema.Option{}, opts...), sema.WithLoader(newOverlay(s.store)))
	return s
}

// Initialize reports what this server serves.
//
// Text synchronization is full rather than incremental. A .tdl file is a
// description of a domain model, the parser is one pass over it, and
// incremental sync would save nothing worth measuring while adding a
// second way for the server's copy of the text to drift from the
// editor's.
func (s *Server) Initialize(context.Context, *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: ptr(true),
				Change:    ptr(protocol.TextDocumentSyncKindFull),
				Save:      &protocol.SaveOptions{IncludeText: ptr(false)},
			},
			DefinitionProvider: protocol.Boolean(true),
			HoverProvider:      protocol.Boolean(true),
			// Formatting is `tdl fmt`, and the outline reads the tree.
			DocumentFormattingProvider: protocol.Boolean(true),
			DocumentSymbolProvider:     protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{Name: "tdl"},
	}, nil
}

func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error { return nil }

func (s *Server) Shutdown(context.Context) error { return nil }

func (s *Server) Exit(context.Context) error { return nil }

func (s *Server) SetTrace(context.Context, *protocol.SetTraceParams) error { return nil }

// DidOpen analyzes a document the editor has opened.
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	item := params.TextDocument
	doc := s.store.open(item.URI, item.Version, item.Text)
	return s.publish(ctx, doc)
}

// DidChange replaces a document's text and re-analyzes it.
//
// Synchronization is full, so the last change carries the whole document,
// which is the arm of the change union this reads. A range change is a
// thing the server never advertised, and applying one as whole text would
// replace the document with a fragment of itself.
//
// A change for a file that was never opened is dropped: the editor and the
// server disagree about what is open, and guessing at the text is worse
// than reporting nothing until the next didOpen.
func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}

	whole, ok := params.ContentChanges[len(params.ContentChanges)-1].(*protocol.TextDocumentContentChangeWholeDocument)
	if !ok {
		return nil
	}

	doc := s.store.change(params.TextDocument.URI, params.TextDocument.Version, whole.Text)
	if doc == nil {
		return nil
	}

	// Every open document is re-analyzed, not just this one: another may
	// import this file, and its diagnostics were computed against the text
	// that just changed.
	s.store.invalidate()
	return s.publishAll(ctx)
}

// DidSave re-analyzes, because a file this document imports may have been
// written by something other than the editor.
func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	if doc := s.store.get(params.TextDocument.URI.FsPath()); doc == nil {
		return nil
	}
	s.store.invalidate()
	return s.publishAll(ctx)
}

// DidClose forgets a document and clears what it published.
//
// An import of it reads the disk from here, so every other open document
// is re-analyzed: one of them may import this file and have been resolving
// it against text that is no longer the server's to know.
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.store.close(params.TextDocument.URI)
	s.store.invalidate()

	if err := s.clear(ctx, params.TextDocument.URI); err != nil {
		return err
	}
	return s.publishAll(ctx)
}

// Definition answers where the name under the cursor was declared.
//
// The answer comes from the index internal/sema records while it resolves,
// so shadowing, the prelude, and `_` imports are decided by the code that
// already decides them and are not restated here.
//
// Three cases return nothing, and all three are an answer rather than a
// failure: the cursor is not on a name, the name did not resolve, or it
// resolved into the prelude, which is embedded and has no file an editor
// can open. docs/design/lsp.md argues the last one.
func (s *Server) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	doc := s.store.get(params.TextDocument.URI.FsPath())
	if doc == nil {
		return nil, nil
	}

	snap := s.store.analyze(doc, s.opts...)
	ref, ok := snap.refs.At(doc.path, snap.index.byteOffset(params.Position))
	if !ok || ref.Target.Filename == "" {
		return nil, nil
	}

	index := s.targetIndex(ref.Target.Filename)
	if index == nil {
		return nil, nil // the prelude, or a file nothing can read
	}

	rng, ok := index.nameSpan(ref.Target, ref.Name)
	if !ok {
		return nil, nil
	}

	return protocol.LocationSlice{{URI: uri.File(ref.Target.Filename), Range: rng}}, nil
}

// targetIndex is the line index of the file a definition points into, or
// nil when there is none to build.
//
// An open document is indexed against the editor's text, and a file that
// is not open is read from disk: a jump has to land on the name, and a
// declaration's column is a byte offset into text the server would
// otherwise not have. Only an absolute path is read, which is what an
// import resolves to; the prelude's name is not a path at all.
func (s *Server) targetIndex(path string) *lineIndex {
	if doc := s.store.get(path); doc != nil {
		return s.store.analyze(doc, s.opts...).index
	}
	if !filepath.IsAbs(path) {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return newLineIndex(string(data))
}

// publishAll re-analyzes every open document.
func (s *Server) publishAll(ctx context.Context) error {
	for _, path := range s.store.paths() {
		doc := s.store.get(path)
		if doc == nil {
			continue
		}
		if err := s.publish(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// publish analyzes a document and sends what it found.
//
// Diagnostics are grouped by the file each one is about rather than sent
// against this document's URI: a problem lowering found in an imported
// file belongs on that file, and an import that fails to resolve belongs
// on the file that wrote it, which is what the position already says.
func (s *Server) publish(ctx context.Context, doc *document) error {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return errNoClient
	}

	snap := s.store.analyze(doc, s.opts...)
	byFile := map[string][]protocol.Diagnostic{}

	// A file that does not parse publishes syntax errors only. Lowering a
	// tree with holes in it reports names that are undefined because the
	// declaration naming them failed to parse, and those are noise.
	for _, e := range snap.errs {
		byFile[e.Pos.Filename] = append(byFile[e.Pos.Filename],
			s.diagnostic(doc, snap, e.Pos.Filename, e.Pos.Line, e.Pos.Col, e.Msg))
	}
	for _, d := range snap.diags {
		byFile[d.Pos.Filename] = append(byFile[d.Pos.Filename],
			s.diagnostic(doc, snap, d.Pos.Filename, d.Pos.Line, d.Pos.Col, d.Msg))
	}

	// This document always publishes, so fixing its last error clears it.
	if _, ok := byFile[doc.path]; !ok {
		byFile[doc.path] = nil
	}

	for path, diags := range byFile {
		sortDiagnostics(diags)
		if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         uri.File(path),
			Diagnostics: diags,
		}); err != nil {
			return err
		}
	}
	return nil
}

// clear publishes an empty list, which is how the protocol says a server
// retracts what it reported about a file.
func (s *Server) clear(ctx context.Context, u uri.URI) error {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return errNoClient
	}

	return client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         u,
		Diagnostics: nil,
	})
}

// diagnostic builds one diagnostic, resolving its range against the text
// of the file it is about.
//
// A position in another file is resolved against that file's text when the
// editor has it open, and reported at the start of its line when it does
// not: the server is not going to read a file off disk to underline a word
// in it.
func (s *Server) diagnostic(doc *document, snap *snapshot, path string, line, col int, msg string) protocol.Diagnostic {
	index := snap.index
	if path != doc.path {
		other := s.store.get(path)
		if other == nil {
			return protocol.Diagnostic{
				Range:    lineRange(line),
				Severity: protocol.DiagnosticSeverityError,
				Source:   protocol.NewOptional(source),
				Message:  protocol.String(msg),
			}
		}
		index = s.store.analyze(other, s.opts...).index
	}

	rng := lineRange(line)
	if off := index.offset(line, col); off >= 0 {
		rng = index.wordSpan(off)
	}
	return protocol.Diagnostic{
		Range:    rng,
		Severity: protocol.DiagnosticSeverityError,
		Source:   protocol.NewOptional(source),
		Message:  protocol.String(msg),
	}
}

// ptr is what an optional protocol field takes: a property that may be
// absent is spelled as a pointer, and a literal has no address.
func ptr[T any](v T) *T { return &v }

// lineRange is the empty range at the start of a 1-based line, for a
// position the server cannot resolve against any text it has.
func lineRange(line int) protocol.Range {
	if line < 1 {
		line = 1
	}
	at := protocol.Position{Line: uint32(line - 1)}
	return protocol.Range{Start: at, End: at}
}

// sortDiagnostics puts them in source order, which is the order a reader
// expects and the order a test can assert on. Lowering reports by pass
// rather than by position, so this is not already true.
func sortDiagnostics(diags []protocol.Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i].Range.Start, diags[j].Range.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Character < b.Character
	})
}
