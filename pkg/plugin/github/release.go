package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/go-github/v67/github"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"github.com/unmango/go/fs/github/repository/release/asset"
	"github.com/unmango/go/option"
	"github.com/unmango/go/rx"
	"github.com/unmango/go/rx/subject"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cache"
	"github.com/unstoppablemango/tdl/pkg/config"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

var ErrMulti = errors.New("multiple release targets specified")

type Release interface {
	tdl.PreReq
	tdl.GeneratorPlugin
	progress.Observable
}

type Option func(*release)

type release struct {
	rx.Subject[progress.Event]
	gh              *github.Client
	owner, repo     string
	asset, release  string
	archiveContents []string
}

// String implements Release.
func (g *release) String() string {
	path := path.Join(
		g.owner, g.repo,
		"releases", "download",
		g.prefixedVersion(), g.asset,
	)

	return fmt.Sprintf("https://github.com/%s", path)
}

// Meta implements Release.
func (g *release) Meta() tdl.Meta {
	return meta.Map{
		"asset":   g.asset,
		"owner":   g.owner,
		"repo":    g.repo,
		"release": g.release,
	}
}

// Ensure implements Release.
func (g *release) Ensure(context.Context) error {
	if path, err := g.lookPath(); err == nil {
		log.Debug("bin found on $PATH", "path", path)
		return nil
	} else {
		return g.cache()
	}
}

// Generator implements tdl.Plugin.
func (g *release) Generator(ctx context.Context, target tdl.Meta) (tdl.Generator, error) {
	if path, err := g.lookPath(); err == nil {
		log.Debug("bin found on $PATH", "path", path)
		return cli.New(path), nil
	}

	if err := g.cache(); err != nil {
		return nil, fmt.Errorf("caching release: %w", err)
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

func (g *release) isArchive() bool {
	return strings.HasSuffix(g.asset, ".tar.gz")
}

func (g release) prefixedVersion() string {
	if strings.HasPrefix(g.release, "v") {
		return g.release
	}

	return fmt.Sprintf("v%s", g.release)
}

func (g release) lookPath() (string, error) {
	if !g.isArchive() {
		return exec.LookPath(g.asset)
	}
	if len(g.archiveContents) == 1 {
		return exec.LookPath(g.archiveContents[0])
	}

	return "", ErrMulti
}

func (r release) cache() error {
	xdgcache, err := config.XdgCache()
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}

	if cache.Exists(xdgcache, r.asset) {
		log.Debug("already cached")
		return nil
	}

	assetfs := asset.NewFs(r.gh, r.owner, r.repo, r.prefixedVersion())
	asset, err := progress.Open(assetfs, r.asset)
	if err != nil {
		return fmt.Errorf("opening release asset: %w", err)
	}

	sub := asset.Subscribe(r)
	defer sub()

	if !r.isArchive() {
		log.Debugf("not an archive: %s", r.asset)
		if config.BinExists(r.asset) {
			log.Debugf("asset is cached: %s", r.asset)
			return nil
		} else {
			log.Debugf("writing bin: %s", r.asset)
			return config.WriteBin(r.asset, asset)
		}
	}

	reader, err := cache.Tee(xdgcache, r.asset, asset)
	if err != nil {
		return fmt.Errorf("writing archive to cache: %w", err)
	}

	gz, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("reading release asset: %w", err)
	}

	tar := tarfs.New(tar.NewReader(gz))
	return afero.Walk(tar, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || info.IsDir() {
				return nil
			}

			name := filepath.Base(path)
			if config.BinExists(name) {
				log.Debugf("bin exists: %s", name)
				return nil
			}

			if e, err := tar.Open(path); err != nil {
				return fmt.Errorf("opening tar entry: %w", err)
			} else {
				log.Debugf("writing bin: %s", name)
				return config.WriteBin(name, e)
			}
		},
	)
}

func NewRelease(asset, name string, options ...Option) Release {
	release := &release{
		Subject: subject.New[progress.Event](),
		owner:   Owner,
		repo:    Repo,
		asset:   asset,
		release: name,
		gh:      github.NewClient(nil),

		archiveContents: []string{},
	}
	option.ApplyAll(release, options)

	return release
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

func WithOptions(options ...Option) Option {
	return func(r *release) {
		option.ApplyAll(r, options)
	}
}
