package lsp

import (
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/sema"
)

// overlay resolves an import against the editor's text when the file is
// open, and against the disk when it is not.
//
// This is what sema.Loader exists for. Lowering touches no filesystem, so
// an editor can hand it buffers nobody has saved and lowering cannot tell
// the difference. Without it, a server would resolve every import against
// text the user has already changed.
type overlay struct {
	store *store
	disk  sema.FSLoader
}

func newOverlay(s *store) *overlay {
	return &overlay{store: s}
}

// Load implements sema.Loader.
//
// Resolution is the filesystem rule sema.FSLoader implements, because that
// is what the spec says a bare import path means, and answering it
// differently here would make an import resolve one way in the editor and
// another on the command line.
func (o *overlay) Load(from, path string) (string, string, error) {
	name := filepath.Join(filepath.Dir(from), path)

	if doc := o.store.get(name); doc != nil {
		return name, doc.text, nil
	}
	return o.disk.Load(from, path)
}
