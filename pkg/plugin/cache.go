package plugin

import (
	"context"
	"log/slog"

	"github.com/google/go-github/v63/github"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

type PluginCache interface {
	PathFor(string) (string, error)
}

// There's a better way to structure these types...
// and I'll figure it out later
type pluginCache struct {
	githubClient
	cache cache.Cache
	log   *slog.Logger
}

func NewCache(client *github.Client, path string, logger *slog.Logger) PluginCache {
	fsCache := cache.NewFsCache(path, logger)
	gh := githubClient{
		client: client,
		cache:  fsCache,
		log:    logger,
	}

	return &pluginCache{
		githubClient: gh,
		cache:        fsCache,
		log:          logger,
	}
}

// PathFor implements PluginCache.
func (c *pluginCache) PathFor(name string) (string, error) {
	p, err := c.cache.Path(name)
	if err == nil {
		return p, nil
	}

	if err = c.populate(context.Background()); err != nil {
		return "", err
	}

	return c.cache.Path(name)
}

func (c *pluginCache) populate(ctx context.Context) error {
	asset, err := c.getLatestAsset(ctx)
	if err != nil {
		return err
	}

	return c.cacheAsset(ctx, asset)
}
