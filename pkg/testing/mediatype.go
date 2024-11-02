package testing

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/unstoppablemango/tdl/pkg/media"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var MediaTypes = []tdl.MediaType{
	media.ApplicationGoogleProtobuf,
	media.ApplicationJson,
	media.ApplicationProtobuf,
	media.ApplicationXProtobuf,
	media.ApplicationXYaml,
	media.ApplicationYaml,
	media.TextJson,
	media.TextYaml,
}

func MediaTypeEntries() []ginkgo.TableEntry {
	entries := make([]ginkgo.TableEntry, len(MediaTypes))
	for i, m := range MediaTypes {
		entries[i] = ginkgo.Entry(nil, m)
	}

	return entries
}
