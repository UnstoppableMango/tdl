package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/roundtrip"
	"github.com/unstoppablemango/tdl/backend/protobuf"
)

// TestCorpus runs testdata/roundtrip/protobuf and smoke here as well as in
// roundtrip's TestCorpus, so the reader's coverage is counted against this
// package: go test counts a package's coverage from its own tests alone.
func TestCorpus(t *testing.T) {
	target := roundtrip.Target{Backend: protobuf.Backend{}, Normalize: protobuf.Normalize}
	roundtrip.Run(t, target, "../../testdata/roundtrip/protobuf", "../../testdata/gen/smoke/source.tdl")
}
