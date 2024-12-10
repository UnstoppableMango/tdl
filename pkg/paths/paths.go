package paths

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	XdgBinHome   = filepath.Join(xdg.BinHome, "ux")
	XdgCacheHome = filepath.Join(xdg.CacheHome, "ux")
)
