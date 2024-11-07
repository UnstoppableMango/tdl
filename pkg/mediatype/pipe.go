package mediatype

import (
	"fmt"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"google.golang.org/protobuf/proto"
)

func PipeRead[
	I c.Pipeline[M, T],
	O c.Pipeline[io.Reader, T],
	M proto.Message, T any,
](pipeline I, media tdl.MediaType, zero func() M) O {
	return func(r io.Reader, t T) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		message := zero()
		if err = Unmarshal(data, message, media); err != nil {
			return fmt.Errorf("unmarshaling message: %w", err)
		}

		return pipeline(message, t)
	}
}
