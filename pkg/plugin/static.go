package plugin

import (
	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/docker"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

var Uml2Ts tdl.Plugin = NewAggregate(
	fromPath{"uml2ts", true, 50},
	github.NewRelease("tdl-linux-amd64.tar.gz", "0.0.30",
		github.WithArchiveContents("uml2ts"),
	),
	docker.New(nil, "ghcr.io/unstoppablemango/uml2ts:v0.0.30"),
)

var Crd2Pulumi tdl.Plugin = NewAggregate(
	github.NewRelease("crd2pulumi-v1.5.4-linux-amd64.tar.gz", "1.5.4",
		github.WithOwner("pulumi"),
		github.WithRepo("crd2pulumi"),
		github.WithArchiveContents("crd2pulumi"),
	),
)

var static = []tdl.Plugin{Uml2Ts, Crd2Pulumi}

func Static() iter.Seq[tdl.Plugin] {
	return slices.Values(static)
}
