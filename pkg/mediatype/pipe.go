package mediatype

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"google.golang.org/protobuf/proto"
)

func PipeRead[
	O c.Pipeline[io.Reader, T],
	I c.Pipeline[M, T],
	M proto.Message, T any,
](pipeline I, media tdl.MediaType, zero func() M) O {
	return func(r io.Reader, t T) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		message := zero() // TODO: Will ProtoReflect().Type().Zero() work?
		if err = Unmarshal(data, message, media); err != nil {
			return fmt.Errorf("unmarshaling message: %w", err)
		}

		return pipeline(message, t)
	}
}

func PipeFs[
	I c.Pipeline[M, T],
	O c.Pipeline[afero.Fs, T],
	M proto.Message, T any,
](pipeline I, path string, zero func() M) O {
	return func(fsys afero.Fs, t T) error {
		media, err := Guess(path)
		if err != nil {
			return fmt.Errorf("guessing media type: %w", err)
		}

		file, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("opening input: %w", err)
		}

		data, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		message := zero()
		if err = Unmarshal(data, message, media); err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		return pipeline(message, t)
	}
}
