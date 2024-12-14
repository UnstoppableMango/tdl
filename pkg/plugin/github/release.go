package github

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/go-github/v67/github"
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
	tdl.Plugin
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

// Prepare implements Release.
func (g *release) Prepare(ctx context.Context) error {
	return g.Ensure(ctx)
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

func (rel release) open() (io.ReadCloser, error) {
	return asset.NewFs(rel.gh,
		rel.owner,
		rel.repo,
		rel.prefixedVersion(),
	).Open(rel.asset)
}

func (rel release) cache() error {
	xdgcache, err := config.XdgCache()
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}

	reader, err := cache.GetOrCreate(xdgcache, rel.asset, rel.open)
	if err != nil {
		return fmt.Errorf("caching asset: %w", err)
	}

	asset := progress.NewReader(reader, reader.Size)
	sub := asset.Subscribe(rel)
	defer sub()

	if rel.isArchive() {
		return rel.extract(asset)
	}

	log.Debugf("not an archive: %s", rel.asset)
	if config.BinExists(rel.asset) {
		log.Debugf("asset is cached: %s", rel.asset)
		return nil
	}

	log.Debugf("writing bin: %s", rel.asset)
	return config.WriteBin(rel.asset, asset)
}

func (rel release) extract(r io.Reader) error {
	xdgbin, err := config.XdgBin()
	if err != nil {
		return fmt.Errorf("opening bin cache: %w", err)
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating zip reader: %w", err)
	}

	return cache.ExtractTarGz(xdgbin, gz)
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
