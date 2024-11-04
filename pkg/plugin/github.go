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
	name, version string
	cache         Config
	client        *github.Client
}

func (g githubRelease) Cache(ctx context.Context) error {
	asset, err := g.getAsset(ctx)
	if err != nil {
		return err
	}

	reader, _, err := g.client.Repositories.DownloadReleaseAsset(ctx,
		GitHubOwner,
		GitHubRepo,
		asset.GetID(),
	)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return g.cache.Cache(g.name, data)
}

func (g githubRelease) getAsset(ctx context.Context) (asset github.ReleaseAsset, err error) {
	release, _, err := g.client.Repositories.GetLatestRelease(ctx,
		GitHubOwner,
		GitHubRepo,
	)
	if err != nil {
		return
	}

	relVer := release.GetName()
	if relVer != g.v() {
		return asset, fmt.Errorf("unsupported release: %s", relVer)
	}

	for _, asset = range release.Assets {
		if asset.GetName() == g.name {
			return asset, nil
		}
	}

	return asset, fmt.Errorf("not found: %s", g.name)
}

func (g githubRelease) v() string {
	return fmt.Sprintf("v%s", g.version)
}
