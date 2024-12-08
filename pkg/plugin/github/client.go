package github

import (
	"net/http"
	"os"

	"github.com/google/go-github/v67/github"
)

const (
	Owner = "UnstoppableMango"
	Repo  = "tdl"
)

var DefaultClient = github.NewClient(http.DefaultClient)

func init() {
	if env, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		DefaultClient = DefaultClient.WithAuthToken(env)
	}
}

type (
	ReleaseAsset      = github.ReleaseAsset
	RepositoryRelease = github.RepositoryRelease
)

func NewUml2Ts(options ...Option) Release {
	options = append(options, WithArchiveContents("uml2ts"))
	return NewRelease("tdl-linux-amd64.tar.gz", "0.0.30", options...)
}
