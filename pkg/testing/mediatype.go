package testing

import (
	"github.com/onsi/ginkgo/v2"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

var MediaTypes = []tdl.MediaType{
	mediatype.ApplicationGoogleProtobuf,
	mediatype.ApplicationJson,
	mediatype.ApplicationProtobuf,
	mediatype.ApplicationXProtobuf,
	mediatype.ApplicationXYaml,
	mediatype.ApplicationYaml,
	mediatype.TextJson,
	mediatype.TextYaml,
}

func MediaTypeEntries() []ginkgo.TableEntry {
	entries := make([]ginkgo.TableEntry, len(MediaTypes))
	for i, m := range MediaTypes {
		entries[i] = ginkgo.Entry(nil, m)
	}

	return entries
}
