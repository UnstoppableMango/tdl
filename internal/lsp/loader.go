package lsp

import (
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/sema"
)

// overlay is a sema.Loader resolving an import against the editor's text
// when the file is open, and against the disk when it is not.
type overlay struct {
	store *store
	disk  sema.FSLoader
}

func newOverlay(s *store) *overlay {
	return &overlay{store: s}
}

// Load implements sema.Loader, resolving paths as sema.FSLoader does so
// the editor and the command line agree.
func (o *overlay) Load(from, path string) (string, string, error) {
	name := filepath.Join(filepath.Dir(from), path)

	if doc := o.store.get(name); doc != nil {
		return name, doc.text, nil
	}
	return o.disk.Load(from, path)
}
