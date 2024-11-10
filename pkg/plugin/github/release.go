package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

type Release interface {
	tdl.Plugin
	cache.Cachable
}

type release struct {
	owner, repo     string
	name, version   string
	client          Client
	archiveContents []string
}

type Option func(*release)

// Generator implements tdl.Plugin.
func (g *release) Generator(target tdl.Target) (tdl.Generator, error) {
	return target.Choose([]tdl.Generator{
		gen.NewCli("uml2ts"),
	})
}

// String implements tdl.Plugin.
func (g *release) String() string {
	path := path.Join(
		g.owner, g.repo,
		"releases", "download",
		g.prefixedVersion(), g.name,
	)

	return fmt.Sprintf("https://github.com/%s", path)
}

func (g release) Cached(c cache.Cacher) bool {
	_, err := c.Reader("")
	return err == nil
}

func (g release) Cache(ctx context.Context, c cache.Cacher) error {
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
		return cache.WriteAll(c, g.name, reader)
	} else {
		return g.extractArchive(c, reader)
	}
}

func (g release) downloadReleaseAsset(ctx context.Context, id int64) (io.Reader, error) {
	reader, _, err := g.client.DownloadReleaseAsset(ctx, g.owner, g.repo, id, http.DefaultClient)
	return reader, err
}

func (g release) extractArchive(c cache.Cacher, reader io.Reader) error {
	if filepath.Ext(g.name) != ".gz" {
		return fmt.Errorf("unsupported archive type: %s", g.name)
	}

	return cache.TarGz(c, reader, g.archiveContents...)
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
		client:  DefaultClient,

		archiveContents: []string{},
	}
	option.ApplyAll(r, options)

	return r
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
