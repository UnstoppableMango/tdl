package run

import (
	"errors"
	"fmt"
	"io"
	"os"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
)

type reader struct {
	io.Reader
	media tdl.MediaType
}

// String implements tdl.Input.
func (r *reader) String() string {
	return fmt.Sprintf("reader: %s", r.media)
}

func (r *reader) MediaType() tdl.MediaType {
	return r.media
}

func Reader(r io.Reader, media tdl.MediaType) tdl.Input {
	return &reader{r, media}
}

func StdinInput(stdin tdl.Stdin) (tdl.Input, error) {
	stat, err := stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, errors.New("nothing on stdin")
	}

	return &reader{stdin, mediatype.ApplicationProtobuf}, nil
}
