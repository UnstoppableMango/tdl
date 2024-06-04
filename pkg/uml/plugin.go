package uml

import "fmt"

var plugins = map[string]string{
	"go":  "uml2go",
	"pcl": "uml2pcl",
	"ts":  "uml2ts",
}

func PluginForTarget(target string) (string, error) {
	if plugin, ok := plugins[target]; ok {
		return plugin, nil
	}

	return "", fmt.Errorf("unsupported target: %s", target)
}
