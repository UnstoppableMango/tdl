package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/google/go-github/v67/github"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"github.com/unmango/go/fs/github/repository/release/asset"
	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type Release interface {
	tdl.GeneratorPlugin
	cache.Cachable
}

type release struct {
	client          Client
	gh              *github.Client
	owner, repo     string
	name, version   string
	archiveContents []string
	progress        progress.ReportFunc
}

type Option func(*release)

// Meta implements Release.
func (g *release) Meta() tdl.Meta {
	return meta.Map{
		"asset":   g.name,
		"owner":   g.owner,
		"repo":    g.repo,
		"version": g.version,
	}
}

// Generator implements tdl.Plugin.
func (g *release) Generator(
	ctx context.Context,
	target tdl.Meta,
) (tdl.Generator, error) {
	if g.isArchive() && len(g.archiveContents) == 1 {
		if path, err := exec.LookPath(g.archiveContents[0]); err == nil {
			log.Debug("bin found on $PATH", "path", path)
			return cli.New(path), nil
		}
	}

	cache, err := cache.Fs()
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	assetfs := afero.NewCacheOnReadFs(
		asset.NewFs(g.gh, g.owner, g.repo, g.prefixedVersion()),
		cache, time.Hour*1, // Always cache
	)
	f, err := progress.Open(assetfs, g.name)
	if err != nil {
		return nil, fmt.Errorf("opening release asset: %w", err)
	}

	if g.progress != nil {
		sub := f.Subscribe(g.progress)
		defer sub()
	}

	if !g.isArchive() {
		return cli.New(g.name), nil
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("reading release asset: %w", err)
	}

	tar := tarfs.New(tar.NewReader(gz))
	bin := afero.NewBasePathFs(afero.NewOsFs(), xdg.BinHome)
	err = afero.Walk(tar, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || info.IsDir() {
				return nil
			}

			if e, err := tar.Open(path); err != nil {
				return err
			} else {
				return afero.WriteReader(bin, path, e)
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("writing binary")
	}

	if len(g.archiveContents) == 1 {
		return cli.New(g.archiveContents[0]), nil
	}

	return nil, errors.New("TODO: some super awesome target matching logic")
}

// Supports implements tdl.Target.
func (g *release) Supports(target tdl.Target) bool {
	return target.String() == "TypeScript" // TODO
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

func (g *release) isArchive() bool {
	return strings.HasSuffix(g.name, ".tar.gz")
}

func (g release) Cached(c cache.Cacher) bool {
	_, err := c.Reader("uml2ts")
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
	if g.progress != nil {
		log.Debug("reporting progress")
		r := progress.NewReader(reader, asset.GetSize())
		sub := r.Subscribe(g.progress)
		defer sub()
		reader = r
	}

	if len(g.archiveContents) == 0 {
		return c.WriteAll(g.name, reader)
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

func WithProgress(report progress.ReportFunc) Option {
	return func(r *release) {
		r.progress = report
	}
}
