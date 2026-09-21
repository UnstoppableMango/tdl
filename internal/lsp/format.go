package lsp

import (
	"context"

	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/ast"
)

// Formatting is `tdl fmt` over the editor's text, returned as one edit
// replacing the whole document.
//
// The formatter owns layout entirely, so the client's tab size and
// whitespace preferences are not read: formatting in an editor has to
// produce what `tdl fmt --check` accepts.
//
// A file that does not parse is left alone. The tree has holes in it, and
// printing one would delete whatever the parser could not read.
func (s *Server) Formatting(_ context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	doc := s.store.get(params.TextDocument.URI.FsPath())
	if doc == nil {
		return nil, nil
	}

	snap := s.store.analyze(doc, s.opts...)
	if snap.file == nil || len(snap.errs) > 0 {
		return nil, nil
	}

	text := snap.index.text
	formatted := ast.Fprint(snap.file)
	if formatted == text {
		return []protocol.TextEdit{}, nil
	}
	return []protocol.TextEdit{{
		Range:   snap.index.span(0, len(text)),
		NewText: formatted,
	}}, nil
}
