package mediatype

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"google.golang.org/protobuf/proto"
)

func PipeInput[
	I c.Pipeline[M, T],
	M proto.Message, T any,
](pipeline I, zero func() M) pipe.Func[tdl.Input, T] {
	return func(i tdl.Input, t T) error {
		next := PipeRead(
			pipeline,
			i.MediaType(),
			zero,
		)

		return next(i, t)
	}
}

func PipeFs[
	I c.Pipeline[M, T],
	M proto.Message, T any,
](pipeline I, path string, zero func() M) pipe.Func[afero.Fs, T] {
	return func(fsys afero.Fs, t T) error {
		media, err := Guess(path)
		if err != nil {
			return fmt.Errorf("guessing media type: %w", err)
		}

		file, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("opening input: %w", err)
		}

		next := PipeRead(
			pipeline,
			media,
			zero,
		)

		return next(file, t)
	}
}

func PipeRead[
	I c.Pipeline[M, T],
	M proto.Message, T any,
](pipeline I, media tdl.MediaType, zero func() M) pipe.Func[io.Reader, T] {
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
