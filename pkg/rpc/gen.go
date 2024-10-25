package rpc

import (
	"context"

	"connectrpc.com/connect"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1/tdlv1alpha1connect"
)

type GenService struct {
	tdlv1alpha1connect.UnimplementedGenServiceHandler
}

func (svc *GenService) Gen(
	ctx context.Context,
	req *connect.Request[tdlv1alpha1.GenRequest],
) (*connect.Response[tdlv1alpha1.GenResponse], error) {
	return svc.UnimplementedGenServiceHandler.Gen(ctx, req)
}

func NewGenService() tdlv1alpha1connect.GenServiceHandler {
	return &GenService{}
}
