package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
	owner, repo   string
	name, version string
	cache         cache.Cacher
	client        Client
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
	panic("unimplemented")
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

	return cache.All(g.cache, g.name, reader)
}

func (g release) downloadReleaseAsset(ctx context.Context, id int64) (io.Reader, error) {
	reader, _, err := g.client.DownloadReleaseAsset(ctx, g.owner, g.repo, id, http.DefaultClient)
	return reader, err
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