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
	O c.Pipeline[tdl.Input, T],
	I c.Pipeline[M, T],
	M proto.Message, T any,
](pipeline I, zero func() M) O {
	return func(i tdl.Input, t T) error {
		next := PipeRead[pipe.Func[io.Reader, T]](
			pipeline,
			i.MediaType(),
			zero,
		)

		return next(i, t)
	}
}

func PipeFs[
	O c.Pipeline[afero.Fs, T],
	I c.Pipeline[M, T],
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

		next := PipeRead[pipe.Func[io.Reader, T]](
			pipeline,
			media,
			zero,
		)

		return next(file, t)
	}
}

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
