package spec

import (
	"bytes"
	"fmt"
	"io"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type readerOptions struct {
	MediaType mediatype.Option
}

type ReaderOption func(*readerOptions)

type reader struct {
	options readerOptions
	spec    *tdlv1alpha1.Spec
	buffer  *bytes.Buffer
}

var defaultOptions = readerOptions{
	MediaType: func() tdl.MediaType {
		return mediatype.ApplicationProtobuf
	},
}

func (r *reader) MediaType() tdl.MediaType {
	return r.options.MediaType()
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

	data, err := ToMediaType(r.MediaType(), r.spec)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	r.buffer = bytes.NewBuffer(data)
	return nil
}

func NewReader(spec *tdlv1alpha1.Spec, options ...ReaderOption) io.Reader {
	return &reader{
		spec:    spec,
		options: Options(options...),
	}
}

func WithMediaType(media tdl.MediaType) ReaderOption {
	return func(o *readerOptions) {
		o.MediaType = func() tdl.MediaType {
			return media
		}
	}
}

func ReadAll(reader io.Reader, options ...ReaderOption) (*tdlv1alpha1.Spec, error) {
	opts := Options(options...)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return FromMediaType(opts.MediaType(), data)
}

func Options(options ...ReaderOption) readerOptions {
	opts := defaultOptions
	option.ApplyAll(&opts, options)
	return opts
}
