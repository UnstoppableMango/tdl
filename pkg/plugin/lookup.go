package plugin

import (
	"fmt"
	"os"
	"path"
)

func LookupPath(name string) (string, error) {
	if dir, found := os.LookupEnv("BIN_DIR"); found {
		bin := path.Join(dir, name)
		if _, err := os.Stat(bin); err != nil {
			return "", err
		}

		return bin, nil
	}

	return "", fmt.Errorf("unable to find plugin: %s", name)
}
