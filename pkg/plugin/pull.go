package plugin

import (
	"context"
	"fmt"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/unmango/go/option"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type PullOptions struct {
	progress progress.TotalFunc
}

func (o PullOptions) github() (opts []github.Option) {
	if o.progress != nil {
		opts = append(opts,
			github.WithProgress(o.progress),
		)
	}

	return
}

type PullOption func(*PullOptions)

func Pull(ctx context.Context, name string, options ...PullOption) error {
	url, err := url.Parse(name)
	if err != nil {
		return fmt.Errorf("matching plugin: %w", err)
	}

	opts := PullOptions{}
	option.ApplyAll(&opts, options)

	if release, err := github.ParseUrl(url, opts.github()...); err == nil {
		return release.Ensure(ctx)
	} else {
		log.Errorf("pulling GitHub release: %s", err)
	}

	log.Debug("unsupported", "url", url)
	return fmt.Errorf("unsupported token: %s", name)
}

func WithProgress(progress progress.TotalFunc) PullOption {
	return func(opts *PullOptions) {
		opts.progress = progress
	}
}
