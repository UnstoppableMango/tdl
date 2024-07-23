package plugin

import (
	"context"
	"fmt"
	"runtime"

	"github.com/google/go-github/v63/github"
	"github.com/unstoppablemango/tdl/pkg/cache"
	"github.com/unstoppablemango/tdl/pkg/logging"
)

var (
	owner = "UnstoppableMango"
	repo  = "tdl"
)

var assetName string

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

type GitHubClient struct {
	client *github.Client
	cache  cache.Cache
}

func NewGitHubClient(client *github.Client, cache cache.Cache) GitHubClient {
	return GitHubClient{client: client, cache: cache}
}

func (c GitHubClient) getReleaseAsset(ctx context.Context) (*github.ReleaseAsset, error) {
	log := logging.FromContext(ctx)

	log.Debug("fetching latest release")
	release, _, err := c.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	log.Debug("searching for asset", "asset", assetName)
	for _, asset := range release.Assets {
		if *asset.Name == assetName {
			return asset, nil
		}

		log.Debug("skipping asset", "asset", asset.Name)
	}

	return nil, fmt.Errorf("unable to find asset %s", assetName)
}

func (c GitHubClient) cacheAsset(ctx context.Context, asset *github.ReleaseAsset) error {
	log := logging.FromContext(ctx)

	log.Debug("downloading release", "asset", asset.Name)
	reader, _, err := c.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, *asset.ID, nil)
	if err != nil {
		return err
	}

	defer reader.Close()

	log.Debug("writing asset to cache")
	return c.cache.Add(assetName, reader)
}
