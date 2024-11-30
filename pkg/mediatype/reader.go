package mediatype

import (
	"bytes"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"google.golang.org/protobuf/proto"
)

type reader struct {
	media   tdl.MediaType
	message proto.Message

	buf *bytes.Buffer
}

// Read implements io.Reader.
func (r *reader) Read(p []byte) (n int, err error) {
	if err = r.ensure(); err != nil {
		return
	}

	return r.buf.Read(p)
}

func (r *reader) ensure() (err error) {
	if r.buf == nil {
		r.buf, err = r.buffer()
	}

	return
}

func (r *reader) buffer() (*bytes.Buffer, error) {
	data, err := Marshal(r.message, r.media)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(data), nil
}

func ProtoReader(message proto.Message, media tdl.MediaType) io.Reader {
	return &reader{
		media:   media,
		message: message,
	}
}
