package plugin

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-github/github"
)

const (
	GitHubOwner = "UnstuppableMango"
	GitHubRepo  = "tdl"
)

type githubRelease struct {
	owner, repo   string
	name, version string
	cache         Cacher
	client        *github.RepositoriesService
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
	release, err := g.getReleaseByTag(ctx, g.v())
	if err != nil {
		return
	}

	// Sanity check
	relVer := release.GetName()
	if relVer != g.v() {
		return asset, fmt.Errorf("unsupported release: %s", relVer)
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

func (g githubRelease) v() string {
	return fmt.Sprintf("v%s", g.version)
}
