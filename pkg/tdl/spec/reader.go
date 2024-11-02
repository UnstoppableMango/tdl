package spec

import (
	"bytes"
	"fmt"
	"io"

	"github.com/unmango/go/option"
	"github.com/unstoppablemango/tdl/pkg/media"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type reader struct {
	spec      *tdlv1alpha1.Spec
	mediatype tdl.MediaType
	buffer    *bytes.Buffer
}

// Read implements io.Reader.
func (r *reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err = r.ensure(); err != nil {
		return len(p), err
	}

	return r.buffer.Read(p)
}

func (r *reader) ensure() error {
	if r.buffer != nil {
		return nil
	}

	data, err := ToMediaType(r.mediatype, r.spec)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	r.buffer = bytes.NewBuffer(data)
	return nil
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

func WithMediaType(media tdl.MediaType) ReaderOption {
	return func(r *reader) {
		r.mediatype = media
	}
}

func ReadAll(reader io.Reader) (*tdlv1alpha1.Spec, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return FromMediaType(media.ApplicationProtobuf, data)
}
