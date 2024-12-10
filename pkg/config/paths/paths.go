package paths

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	XdgBinHome   = xdg.BinHome
	XdgCacheHome = filepath.Join(xdg.CacheHome, "ux")
)
