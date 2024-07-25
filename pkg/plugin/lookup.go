package plugin

import (
	"fmt"
	"os"
	"path"
)

var plugins = map[string]string{
	"go":  "uml2go",
	"pcl": "uml2pcl",
	"ts":  "uml2ts",
}

func ForTarget(target string) (string, error) {
	if plugin, ok := plugins[target]; ok {
		return plugin, nil
	}

	return "", fmt.Errorf("unsupported target: %s", target)
}

func LookupPath(name string) (string, error) {
	dir, found := os.LookupEnv("BIN_DIR")
	if !found {
		return "", fmt.Errorf("unable to find plugin: %s", name)
	}

	bin := path.Join(dir, name)
	if _, err := os.Stat(bin); err != nil {
		return "", err
	}

	return bin, nil
}
