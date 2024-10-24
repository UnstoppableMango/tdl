package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/google/go-github/v66/github"
	"github.com/unstoppablemango/tdl/pkg/cache"
)

var (
	owner     = "UnstoppableMango"
	repo      = "tdl"
	assetName = fmt.Sprintf("tdl_%s_%s.tar.gz",
		runtime.GOOS,
		runtime.GOARCH,
	)
)

type githubClient struct {
	client *github.Client
	cache  cache.Cache
	log    *slog.Logger
}

func (c githubClient) getLatestAsset(ctx context.Context) (*github.ReleaseAsset, error) {
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

func (c githubClient) cacheAsset(ctx context.Context, asset *github.ReleaseAsset) error {
	log := c.log.With("asset", asset.Name)

	log.Debug("downloading release")
	reader, _, err := c.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, *asset.ID, http.DefaultClient)
	if err != nil {
		return err
	}
	if reader == nil {
		return errors.New("github redirects not supported")
	}

	defer reader.Close()

	gzipStream, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	tarFile := tar.NewReader(gzipStream)
	h, err := tarFile.Next()
	for err != io.EOF {
		if err != nil {
			return err
		}

		log.Debug("caching tar entry", "name", h.Name)
		err = c.cache.Add(h.Name, tarFile)
		if err != nil {
			return err
		}

		h, err = tarFile.Next()
	}

	log.Debug("finished caching asset")
	return nil
}
