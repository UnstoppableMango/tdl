package plugin

import (
	"context"
	"fmt"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

type GitHubRelease interface {
	tdl.Plugin
	Cached() bool
	Cache(context.Context) error
}

type githubRelease struct {
	owner, repo   string
	name, version string
	cache         Cacher
	client        github.Client
}

// Cached implements GitHubRelease.
func (g *githubRelease) Cached() bool {
	panic("unimplemented")
}

// Generator implements tdl.Plugin.
func (g *githubRelease) Generator(tdl.Target) (tdl.Generator, error) {
	panic("unimplemented")
}

// String implements tdl.Plugin.
func (g *githubRelease) String() string {
	panic("unimplemented")
}

func (g githubRelease) Cache(ctx context.Context) error {
	asset, err := g.getAsset(ctx)
	if err != nil {
		return err
	}

	reader, err := g.downloadReleaseAsset(ctx, asset.GetID())
	if err != nil {
		return err
	}

	return CacheAll(g.cache, g.name, reader)
}

func (g githubRelease) downloadReleaseAsset(ctx context.Context, id int64) (io.Reader, error) {
	reader, _, err := g.client.DownloadReleaseAsset(ctx, g.owner, g.repo, id)
	return reader, err
}

func (g githubRelease) getAsset(ctx context.Context) (asset github.ReleaseAsset, err error) {
	release, err := g.getReleaseByTag(ctx, g.prefixedVersion())
	if err != nil {
		return
	}

	for _, asset = range release.Assets {
		if asset.GetName() == g.name {
			return
		}
	}

	return asset, fmt.Errorf("not found: %s", g.name)
}

func (g githubRelease) getReleaseByTag(ctx context.Context, tag string) (*github.RepositoryRelease, error) {
	release, _, err := g.client.GetReleaseByTag(ctx, g.owner, g.repo, tag)
	return release, err
}

func (g githubRelease) prefixedVersion() string {
	return fmt.Sprintf("v%s", g.version)
}

func NewGitHubRelease(client github.Client, name, version string) GitHubRelease {
	return &githubRelease{
		owner:   github.Owner,
		repo:    github.Repo,
		name:    name,
		version: version,
		cache:   XdgConfig,
		client:  client,
	}
}
