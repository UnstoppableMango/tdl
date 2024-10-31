package gen

import (
	"context"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"github.com/unstoppablemango/tdl/pkg/gen/memory"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/pipe"
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
	sink := memory.NewPipe()
	if err := svc.generator(req.Msg.Spec, sink); err != nil {
		return nil, fmt.Errorf("invoking generator: %w", err)
	}

	units, err := pipe.Map(sink, readUnit)
	if err != nil {
		return nil, fmt.Errorf("mapping units: %w", err)
	}

	res := &tdlv1alpha1.GenResponse{
		Output: units,
	}

	return connect.NewResponse(res), nil
}

func NewHandler(generator tdl.Gen) tdlv1alpha1connect.GenServiceHandler {
	return &connectService{generator: generator}
}

func readUnit(s string, r io.Reader) (*tdlv1alpha1.Unit, error) {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading from sink: %w", err)
	}

	return &tdlv1alpha1.Unit{
		Generated: bytes,
	}, nil
}
