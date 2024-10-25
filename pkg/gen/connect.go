package gen

import (
	"bytes"
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1/tdlv1alpha1connect"
)

type connectService struct {
	tdlv1alpha1connect.UnimplementedGenServiceHandler

	generator tdl.Gen
}

func (svc *connectService) Gen(
	ctx context.Context,
	req *connect.Request[tdlv1alpha1.GenRequest],
) (*connect.Response[tdlv1alpha1.GenResponse], error) {
	buf := &bytes.Buffer{}
	if err := svc.generator(req.Msg.Spec, buf); err != nil {
		return nil, fmt.Errorf("invoking generator: %w", err)
	}

	res := &tdlv1alpha1.GenResponse{
		Output: map[string]*tdlv1alpha1.Unit{
			"default": {Generated: buf.Bytes()},
		},
	}

	return connect.NewResponse(res), nil
}

func NewHandler(generator tdl.Gen) tdlv1alpha1connect.GenServiceHandler {
	return &connectService{generator: generator}
}
