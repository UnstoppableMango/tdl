package github

import (
	"context"
	"io"
	"net/http"

	"github.com/google/go-github/v66/github"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

const (
	Owner = "UnstoppableMango"
	Repo  = "tdl"
)

var DefaultClient = NewClient(
	github.NewClient(http.DefaultClient),
)

type Client interface {
	BaseURL() string
	DownloadReleaseAsset(ctx context.Context, owner string, repo string, id int64, followRedirectsClient *http.Client) (io.ReadCloser, string, error)
	GetReleaseByTag(ctx context.Context, owner string, repo string, tag string) (*github.RepositoryRelease, *github.Response, error)
}

type (
	ReleaseAsset      = github.ReleaseAsset
	RepositoryRelease = github.RepositoryRelease
)

type client struct {
	*github.RepositoriesService
	gh *github.Client
}

// BaseURL implements Client.
func (c *client) BaseURL() string {
	return c.gh.BaseURL.String()
}

func NewClient(github *github.Client) Client {
	return &client{github.Repositories, github}
}

func NewUml2Ts(options ...Option) tdl.Plugin {
	options = append(options, WithArchiveContents("uml2ts"))
	return NewRelease("tdl-linux-amd64.tar.gz", "0.0.29", options...)
}
