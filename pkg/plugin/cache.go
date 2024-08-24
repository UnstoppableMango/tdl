package plugin

import (
	"context"
	"log/slog"
	"os"
	"path"

	"github.com/google/go-github/v64/github"
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
	if bin, ok := c.searchEnv(name, "BIN_DIR", "TDL_BIN"); ok {
		c.log.Info("using plugin found in env", "bin", bin)
		return bin, nil
	}

	if bin, err := c.cache.Path(name); err == nil {
		c.log.Info("using plugin cached at path", "path", bin)
		return bin, nil
	}

	if err := c.populate(context.Background()); err != nil {
		c.log.Error("failed to populate the cache", "err", err)
		return "", err
	}

	c.log.Debug("retrying cache", "name", name)
	return c.cache.Path(name)
}

func (c *pluginCache) searchEnv(name string, envs ...string) (string, bool) {
	for _, env := range envs {
		c.log.Debug("searching env", "env", env)
		if bin, ok := c.fromEnv(name, env); ok {
			return bin, true
		}
	}

	c.log.Debug("not found in envs", "name", name, "envs", envs)
	return "", false
}

func (c *pluginCache) fromEnv(name, env string) (string, bool) {
	binDir, ok := os.LookupEnv(env)
	if !ok {
		c.log.Debug("unable to find env", "env", env)
		return "", false
	}

	bin := path.Join(binDir, name)
	_, err := os.Stat(bin)

	c.log.Debug("found env",
		"env", env,
		"dir", binDir,
		"bin", bin,
		"err", err,
	)

	return bin, err == nil
}

func (c *pluginCache) populate(ctx context.Context) error {
	asset, err := c.getLatestAsset(ctx)
	if err != nil {
		return err
	}

	return c.cacheAsset(ctx, asset)
}
