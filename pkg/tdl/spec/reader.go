package spec

import (
	"fmt"
	"io"

	"github.com/unmango/go/option"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/media"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type reader struct {
	spec      *tdlv1alpha1.Spec
	mediatype tdl.MediaType
}

// Read implements io.Reader.
func (r *reader) Read(p []byte) (n int, err error) {
	p, err = ToMediaType(r.mediatype, r.spec)
	if err != nil {
		return 0, fmt.Errorf("marshaling: %w", err)
	}

	return len(p), nil
}

type ReaderOption func(*reader)

func NewReader(spec *tdlv1alpha1.Spec, options ...ReaderOption) io.Reader {
	reader := &reader{spec, media.ApplicationProtobuf}
	option.ApplyAll(reader, options)

	return reader
}
