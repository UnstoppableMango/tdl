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
	bytes     []byte
	offset    int
	err       error
}

// Read implements io.Reader.
func (r *reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.bytes == nil {
		r.bytes, r.err = ToMediaType(r.mediatype, r.spec)
	}
	if r.err != nil {
		return len(p), fmt.Errorf("marshaling: %w", err)
	}

	var i int
	for i = 0; i < len(p); i++ {
		if i+r.offset >= len(r.bytes) {
			return i, io.EOF
		}

		p[i] = r.bytes[i+r.offset]
	}

	r.offset = i + r.offset
	return i, nil
}

type ReaderOption func(*reader)

func NewReader(spec *tdlv1alpha1.Spec, options ...ReaderOption) io.Reader {
	reader := &reader{
		spec:      spec,
		mediatype: media.ApplicationProtobuf,
	}
	option.ApplyAll(reader, options)

	return reader
}
