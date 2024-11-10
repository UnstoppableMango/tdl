package input

import (
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

type reader struct {
	io.Reader
	media tdl.MediaType
}

func (r *reader) MediaType() tdl.MediaType {
	return r.media
}

func Reader(r io.Reader, media tdl.MediaType) tdl.Input {
	return &reader{r, media}
}

func Stdin(stdin io.Reader) tdl.Input {
	return &reader{stdin, mediatype.ApplicationProtobuf}
}
