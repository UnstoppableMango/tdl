package github

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/unmango/go/fs/github/ghpath"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

var SupportedSchemes = []string{
	"github",
	"https",
	"http",
}

const Parser parser = "GitHub"

type parser string

func (parser) Parse(token string) (tdl.Plugin, error) {
	if url, err := url.Parse(token); err != nil {
		return nil, fmt.Errorf("not a url: %w", err)
	} else {
		return ParseUrl(url)
	}
}

var _ tdl.Parser[tdl.Plugin] = Parser

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
