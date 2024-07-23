package plugin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v63/github"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

type PluginCache interface {
	PathFor(string) (string, error)
}

type pluginCache struct {
	cache  cache.Cache
	client *github.Client
	log    *slog.Logger
}

func NewCache(client *github.Client, path string, logger *slog.Logger) PluginCache {
	return &pluginCache{
		cache:  cache.NewFsCache(path, logger),
		client: client,
		log:    logger,
	}
}

// PathFor implements PluginCache.
func (c *pluginCache) PathFor(name string) (string, error) {
	cached, err := c.cache.Get(name)
	if err == nil {
		cached.Close()
		c.cache.Path(name)
	}

	ctx := context.Background()
	c.populate(ctx)

	// TODO: Ensure file exists
	return c.cache.Path(name), nil
}

func (c *pluginCache) populate(ctx context.Context) error {
	asset, err := c.getLatestAsset(ctx)
	if err != nil {
		return err
	}

	if err = c.cacheAsset(ctx, asset); err != nil {
		return err
	}

	// TODO: Extract archive
	return nil
}

func (c pluginCache) getLatestAsset(ctx context.Context) (*github.ReleaseAsset, error) {
	c.log.Debug("fetching latest release")
	release, _, err := c.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	c.log.Debug("searching for asset", "asset", assetName)
	for _, asset := range release.Assets {
		if *asset.Name == assetName {
			return asset, nil
		}

		c.log.Debug("skipping asset", "asset", asset.Name)
	}

	return nil, fmt.Errorf("unable to find asset %s", assetName)
}

func (c pluginCache) cacheAsset(ctx context.Context, asset *github.ReleaseAsset) error {
	c.log.Debug("downloading release", "asset", asset.Name)
	reader, _, err := c.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, *asset.ID, nil)
	if err != nil {
		return err
	}

	defer reader.Close()

	c.log.Debug("writing asset to cache")
	return c.cache.Add(assetName, reader)
}
