package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/google/go-github/v67/github"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"github.com/unmango/go/fs/github/ghpath"
	"github.com/unmango/go/fs/github/repository/release/asset"
	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

var SupportedSchemes = []string{
	"github",
	"https",
	"http",
}

type Release interface {
	tdl.PreReq
	tdl.GeneratorPlugin
}

type Option func(*release)

type release struct {
	client          Client
	gh              *github.Client
	owner, repo     string
	name, version   string
	archiveContents []string
	progress        progress.ReportFunc
}

// Ensure implements Release.
func (g *release) Ensure(context.Context) error {
	if g.isArchive() && len(g.archiveContents) == 1 {
		if path, err := exec.LookPath(g.archiveContents[0]); err == nil {
			log.Debug("bin found on $PATH", "path", path)
			return nil
		}
	}

	cache, err := cache.Fs()
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}

	assetfs := afero.NewCacheOnReadFs(
		asset.NewFs(g.gh, g.owner, g.repo, g.prefixedVersion()),
		cache, time.Hour*1, // Always cache
	)
	f, err := progress.Open(assetfs, g.name)
	if err != nil {
		return fmt.Errorf("opening release asset: %w", err)
	}

	if g.progress != nil {
		sub := f.Subscribe(g.progress)
		defer sub()
	}

	if !g.isArchive() {
		log.Debug("not an archive: %s")
		return nil
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading release asset: %w", err)
	}

	tar := tarfs.New(tar.NewReader(gz))
	bin := afero.NewBasePathFs(afero.NewOsFs(), xdg.BinHome)
	return afero.Walk(tar, "",
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
}

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
	return meta.HasValue(target.Meta(),
		meta.WellKnown.Lang,
		meta.Lang.TypeScript,
	)
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

func (g release) prefixedVersion() string {
	if strings.HasPrefix(g.version, "v") {
		return g.version
	}

	return fmt.Sprintf("v%s", g.version)
}

func ParseUrl(url *url.URL, options ...Option) (Release, error) {
	if !slices.Contains(SupportedSchemes, url.Scheme) {
		return nil, fmt.Errorf("unsupported scheme: %s", url)
	}

	path, err := ghpath.ParseUrl(url.String())
	if err != nil {
		return nil, fmt.Errorf("ghpath: %w", err)
	}

	asset, err := ghpath.ParseAsset(path)
	if err != nil {
		return nil, fmt.Errorf("ghpath: %w", err)
	}

	return NewRelease(asset.Asset, asset.Release,
		WithRepository(asset.Owner, asset.Repository),
		WithOptions(options...),
	), nil
}

func NewRelease(name, version string, options ...Option) Release {
	release := &release{
		owner:   Owner,
		repo:    Repo,
		name:    name,
		version: version,
		client:  DefaultClient,
		gh:      github.NewClient(nil),

		archiveContents: []string{},
	}
	option.ApplyAll(release, options)

	return release
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

func WithOptions(options ...Option) Option {
	return func(r *release) {
		option.ApplyAll(r, options)
	}
}
