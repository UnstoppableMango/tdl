package gen

import (
	"path/filepath"
	"strings"
)

// rooted reports whether p names a place apart from any directory it might
// be joined to: absolute, or on Windows, starting at a drive's root or
// naming a drive. A path beginning with a slash is rooted on every OS, so a
// model or a backend means the same by it everywhere.
func rooted(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) || filepath.VolumeName(p) != ""
}
