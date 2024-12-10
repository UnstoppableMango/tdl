package plugin

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

func ParseToken(token string) (tdl.Plugin, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("empty token")
	}
	if url, err := url.Parse(token); err == nil {
		return ParseUrl(url)
	} else {
		log.Debugf("parse token: not a url: %s", token)
	}

	return nil, fmt.Errorf("unsupported token: %s", token)
}

func ParseUrl(url *url.URL) (tdl.Plugin, error) {
	if release, err := github.ParseUrl(url); err == nil {
		return release, nil
	} else {
		log.Debugf("parsing github url: %s", err)
	}

	return nil, fmt.Errorf("unsupported url: %s", url)
}
