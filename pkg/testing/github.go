package testing

import (
	"net/http"

	"github.com/google/go-github/v69/github"
)

func NewGitHubClient() *github.Client {
	return github.NewClient(http.DefaultClient)
}
