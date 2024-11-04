package testing

import (
	"net/http"

	"github.com/google/go-github/github"
)

func NewGitHubClient() *github.Client {
	return github.NewClient(http.DefaultClient)
}
