package plugin

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/google/go-github/v63/github"
)

var assetName string
var plugins = map[string]string{
	"go":  "uml2go",
	"pcl": "uml2pcl",
	"ts":  "uml2ts",
}

func init() {
	var os, ext string
	switch runtime.GOOS {
	case "linux":
		os = "Linux"
		ext = "tar.gz"
	case "darwin":
		os = "Darwin"
		ext = "tar.gz"
	case "windows":
		os = "Windows"
		ext = "zip"
	}

	assetName = fmt.Sprintf("tdl_%s_%s.%s",
		os,
		runtime.GOARCH,
		ext,
	)
}

func ForTarget(target string) (string, error) {
	if plugin, ok := plugins[target]; ok {
		return plugin, nil
	}

	return "", fmt.Errorf("unsupported target: %s", target)
}

func Download(ctx context.Context, client *github.Client, target string) (string, error) {
	release, _, err := client.Repositories.GetLatestRelease(ctx, "UnstoppableMango", "tdl")
	if err != nil {
		return "", err
	}

	var asset *github.ReleaseAsset = nil
	for _, x := range release.Assets {
		if *x.Name == assetName {
			asset = x
		}
	}
	if asset == nil {
		return "", fmt.Errorf("unable to find asset %s", assetName)
	}

	return "", errors.New("TODO")
}
