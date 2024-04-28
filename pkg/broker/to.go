package broker

import (
	"context"
	"io"

	"connectrpc.com/connect"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type ToRequest = connect.Request[tdl.ToRequest]
type ToServerStream = connect.ServerStreamForClient[tdl.ToResponse]

type toStreamReader struct {
	*ToServerStream
}

func (r *toStreamReader) Read(p []byte) (n int, err error) {
	if !r.Receive() {
		return 0, r.Err()
	}

	data := r.Msg().GetData()
	return copy(p, data), nil
}

var _ io.Reader = &toStreamReader{}

func (b *broker) To(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	stream, err := b.client.To(ctx, &ToRequest{
		Msg: &tdl.ToRequest{Spec: spec},
	})
	if err != nil {
		return err
	}

	reader := toStreamReader{ToServerStream: stream}
	_, err = io.Copy(writer, &reader)

	return err
}
