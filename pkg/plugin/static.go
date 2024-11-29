package plugin

import (
	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/docker"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

var Uml2Ts tdl.Plugin = NewAggregate(
	fromPath{"uml2ts", 50},
	github.NewRelease("tdl-linux-amd64.tar.gz", "v0.0.30",
		github.WithArchiveContents("uml2ts"),
	),
	docker.New(nil, "ghcr.io/unstoppablemango/uml2ts:v0.0.30"),
)

var static = []tdl.Plugin{Uml2Ts}

func Static() iter.Seq[tdl.Plugin] {
	return slices.Values(static)
}
