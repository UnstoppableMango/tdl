package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/log"
	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type Release interface {
	tdl.Plugin
	Cached() bool
	Cache(context.Context) error
}

type release struct {
	owner, repo     string
	name, version   string
	cache           cache.Cacher
	client          Client
	archiveContents []string
}

type Option func(*release)

// Cached implements Release.
func (g *release) Cached() bool {
	panic("unimplemented")
}

// Generator implements tdl.Plugin.
func (g *release) Generator(tdl.Target) (tdl.Generator, error) {
	panic("unimplemented")
}

// String implements tdl.Plugin.
func (g *release) String() string {
	baseUrl := g.client.BaseURL()
	url, err := url.JoinPath(baseUrl, "")
	if err != nil {
		log.Error(err)
		return baseUrl
	}

	return url
}

func (g release) Cache(ctx context.Context) error {
	asset, err := g.getAsset(ctx)
	if err != nil {
		return err
	}

	reader, err := g.downloadReleaseAsset(ctx, asset.GetID())
	if err != nil {
		return err
	}
	if reader == nil {
		return fmt.Errorf("reader was nil")
	}

	if len(g.archiveContents) == 0 {
		return cache.All(g.cache, g.name, reader)
	} else {
		return g.extractArchive(reader)
	}
}

func (g release) downloadReleaseAsset(ctx context.Context, id int64) (io.Reader, error) {
	reader, _, err := g.client.DownloadReleaseAsset(ctx, g.owner, g.repo, id, http.DefaultClient)
	return reader, err
}

func (g release) extractArchive(reader io.Reader) error {
	if filepath.Ext(g.name) != ".gz" {
		return fmt.Errorf("unsupported archive type: %s", g.name)
	}

	gz, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	var name string
	tar := tar.NewReader(gz)
	header, err := tar.Next()
	for (err == nil || errors.Is(err, io.EOF)) && header != nil {
		name = header.Name
		if !slices.Contains(g.archiveContents, name) {
			log.Debug("skipping archive entry", "name", name)
			continue
		}

		err = cache.All(g.cache, name, tar)
		header, err = tar.Next()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("extracting %s: %w", name, err)
	}

	return nil
}

func (g release) getAsset(ctx context.Context) (asset *ReleaseAsset, err error) {
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

func (g release) getReleaseByTag(ctx context.Context, tag string) (*RepositoryRelease, error) {
	release, _, err := g.client.GetReleaseByTag(ctx, g.owner, g.repo, tag)
	return release, err
}

func (g release) prefixedVersion() string {
	return fmt.Sprintf("v%s", g.version)
}

func NewRelease(name, version string, options ...Option) Release {
	r := &release{
		owner:   Owner,
		repo:    Repo,
		name:    name,
		version: version,
		cache:   cache.XdgConfig,
		client:  DefaultClient,

		archiveContents: []string{},
	}
	option.ApplyAll(r, options)

	return r
}

func WithCache(cache cache.Cacher) Option {
	return func(r *release) {
		r.cache = cache
	}
}

func WithClient(client Client) Option {
	return func(r *release) {
		r.client = client
	}
}

func WithOwner(owner string) Option {
	return func(r *release) {
		r.owner = owner
	}
}

func WithRepo(repo string) Option {
	return func(r *release) {
		r.repo = repo
	}
}

func WithRepository(owner, repo string) Option {
	return func(r *release) {
		r.owner = owner
		r.repo = repo
	}
}

func WithArchiveContents(path ...string) Option {
	return func(r *release) {
		r.archiveContents = append(r.archiveContents, path...)
	}
}
