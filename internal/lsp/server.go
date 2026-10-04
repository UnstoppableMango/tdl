// Package lsp serves the Language Server Protocol for TDL: diagnostics,
// go to definition, hover, document symbols, and formatting.
//
// See docs/design/lsp.md and docs/design/lsp-plan.md.
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

// source names this server in each diagnostic.
const source = "tdl"

// errNoClient answers a request whose context carries no client, which
// Serve always supplies.
var errNoClient = errors.New("lsp: no client in context")

// Server answers protocol requests about the documents an editor has open.
// A request it does not implement gets "method not found" from the
// embedded protocol.UnimplementedServer.
type Server struct {
	protocol.UnimplementedServer

	store *store
	opts  []sema.Option
}

// NewServer returns a server analyzing every document with opts. The
// server appends its own loader, so an import of an open document resolves
// to its unsaved text. The client comes from each request's context.
func NewServer(opts ...sema.Option) *Server {
	s := &Server{store: newStore()}
	s.opts = append(append([]sema.Option{}, opts...), sema.WithLoader(newOverlay(s.store)))
	return s
}

// Initialize reports what this server serves. Text synchronization is
// full: a .tdl file is small, and incremental sync would add a way for the
// server's copy to drift from the editor's.
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

// DidChange replaces a document's text and re-analyzes it. With full sync
// the last change carries the whole document; a range change was never
// advertised and is ignored. A change for a file never opened is dropped.
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

	// Another open document may import this one.
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

// DidClose forgets a document and clears what it published. Every other
// open document is re-analyzed, since an import of this one reads the disk
// from here on.
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.store.close(params.TextDocument.URI)
	s.store.invalidate()

	if err := s.clear(ctx, params.TextDocument.URI); err != nil {
		return err
	}
	return s.publishAll(ctx)
}

// Definition answers where the name under the cursor was declared, read
// from the index sema records while it resolves. It returns nothing when
// the cursor is not on a name, the name did not resolve, or it resolved
// into the embedded prelude.
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
// nil when there is none. A file that is not open is read from disk,
// because a declaration's column is a byte offset into its text. Only an
// absolute path is read; the prelude's name is not one.
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

// publish analyzes a document and sends its diagnostics, grouped by the
// file each position names, so a problem in an imported file is reported
// on that file.
func (s *Server) publish(ctx context.Context, doc *document) error {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return errNoClient
	}

	snap := s.store.analyze(doc, s.opts...)
	byFile := map[string][]protocol.Diagnostic{}

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

// clear retracts what was reported about a file by publishing an empty
// list.
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

// diagnostic builds one diagnostic, ranging over the word at its
// position. A position in a file that is not open is reported at the start
// of its line.
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

// ptr returns a pointer to v, for optional protocol fields.
func ptr[T any](v T) *T { return &v }

// lineRange is the empty range at the start of a 1-based line.
func lineRange(line int) protocol.Range {
	if line < 1 {
		line = 1
	}
	at := protocol.Position{Line: uint32(line - 1)}
	return protocol.Range{Start: at, End: at}
}

// sortDiagnostics puts diagnostics in source order. Lowering reports them
// by pass.
func sortDiagnostics(diags []protocol.Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i].Range.Start, diags[j].Range.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Character < b.Character
	})
}
