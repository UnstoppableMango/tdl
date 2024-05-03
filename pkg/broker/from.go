package broker

import (
	"context"
	"io"

	"connectrpc.com/connect"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type FromClientStream = connect.ClientStreamForClient[tdl.FromRequest, tdl.FromResponse]

type fromStreamWriter struct {
	*FromClientStream
}

func (w *fromStreamWriter) Write(p []byte) (n int, err error) {
	req := tdl.FromRequest{Data: p}

	err = w.Send(&req)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

var _ io.Writer = &fromStreamWriter{}

func (b *broker) From(ctx context.Context, reader io.Reader) (*uml.Spec, error) {
	stream := b.client.From(ctx)
	writer := fromStreamWriter{FromClientStream: stream}
	if _, err := io.Copy(&writer, reader); err != nil {
		return nil, err
	}

	res, err := stream.CloseAndReceive()
	if err != nil {
		return nil, err
	}

	return res.Msg.Spec, nil
}
