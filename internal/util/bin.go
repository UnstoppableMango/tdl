package util

import (
	"context"
	"path/filepath"
)

func BinPath(name string) string {
	root, err := GitRoot(context.Background())
	if err != nil {
		panic(err)
	}

	return filepath.Join(root, "bin", name)
}
