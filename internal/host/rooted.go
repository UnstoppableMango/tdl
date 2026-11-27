package host

import (
	"os"
	"path/filepath"
	"strings"
)

// Rooted reports whether an out directive's path names a place apart from
// the file declaring it, rather than one relative to that file: absolute,
// starting with a slash on every OS, or on Windows starting at a drive's
// root or naming a drive.
func Rooted(p string) bool {
	return filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.HasPrefix(p, "/") ||
		(p != "" && os.IsPathSeparator(p[0]))
}
