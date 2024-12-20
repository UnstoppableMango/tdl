package internal

import (
	"os"
	"path/filepath"
)

// Cwd cleans and roots [op] or retrieves the current directory
func Cwd(op string) (cwd string, err error) {
	if cwd = op; op == "" {
		if cwd, err = os.Getwd(); err != nil {
			return
		}
	}

	if !filepath.IsAbs(cwd) {
		if cwd, err = filepath.Abs(cwd); err != nil {
			return
		}
	}

	return
}
