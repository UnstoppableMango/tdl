package broker

import (
	"context"
	"io"

	"connectrpc.com/connect"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type GenRequest = connect.Request[tdl.GenRequest]
type GenServerStream = connect.ServerStreamForClient[tdl.GenResponse]

type genStreamReader struct {
	*GenServerStream
}

func (r *genStreamReader) Read(p []byte) (n int, err error) {
	if !r.Receive() {
		return 0, r.Err()
	}

	data := r.Msg().GetData()
	return copy(p, data), nil
}

var _ io.Reader = &genStreamReader{}

func (b *broker) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	stream, err := b.client.Gen(ctx, &GenRequest{
		Msg: &tdl.GenRequest{Spec: spec},
	})
	if err != nil {
		return err
	}

	reader := genStreamReader{GenServerStream: stream}
	_, err = io.Copy(writer, &reader)

	return err
}
