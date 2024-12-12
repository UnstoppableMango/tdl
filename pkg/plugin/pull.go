package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/unmango/go/option"
	"github.com/unmango/go/rx"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
	"github.com/unstoppablemango/tdl/pkg/progress"
)

type PullOptions struct {
	progress rx.Observer[progress.Event]
}

type PullOption func(*PullOptions)

func Pull(ctx context.Context, plugin tdl.Plugin, options ...PullOption) error {
	opts := PullOptions{}
	option.ApplyAll(&opts, options)

	if opts.progress != nil {
		log.Debug("subscribing to progress events")
		sub := Observe(plugin).Subscribe(opts.progress)
		defer sub()
	}

	// return prereq.Ensure(ctx)
	return errors.New("TODO: pull")
}

func PullToken(ctx context.Context, name string, options ...PullOption) error {
	url, err := url.Parse(name)
	if err != nil {
		return fmt.Errorf("matching plugin: %w", err)
	}

	opts := PullOptions{}
	option.ApplyAll(&opts, options)

	if release, err := github.ParseUrl(url); err == nil {
		return Pull(ctx, release, options...)
	} else {
		log.Errorf("pulling GitHub release: %s", err)
	}

	log.Debug("unsupported", "url", url)
	return fmt.Errorf("unsupported token: %s", name)
}

func WithProgress(progress rx.Observer[progress.Event]) PullOption {
	return func(opts *PullOptions) {
		opts.progress = progress
	}
}
