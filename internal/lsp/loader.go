package lsp

import (
	"github.com/unstoppablemango/tdl/internal/sema"
)

// newOverlay returns a sema.Loader resolving an import against the
// editor's text when the file is open, and against the disk when it is
// not, so the editor and the command line agree.
func newOverlay(s *store) sema.Loader {
	return sema.Overlay{Text: func(name string) (string, bool) {
		if doc := s.get(name); doc != nil {
			return doc.text, true
		}
		return "", false
	}}
}
