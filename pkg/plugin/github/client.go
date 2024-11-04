package github

import (
	"context"
	"io"

	"github.com/google/go-github/github"
)

const (
	Owner = "UnstoppableMango"
	Repo  = "tdl"
)

type Client interface {
	DownloadReleaseAsset(ctx context.Context, owner string, repo string, id int64) (io.ReadCloser, string, error)
	GetReleaseByTag(ctx context.Context, owner string, repo string, tag string) (*github.RepositoryRelease, *github.Response, error)
}

type (
	ReleaseAsset      = github.ReleaseAsset
	RepositoryRelease = github.RepositoryRelease
)

type client struct {
	*github.RepositoriesService
}

func NewClient(github *github.Client) Client {
	return &client{github.Repositories}
}
