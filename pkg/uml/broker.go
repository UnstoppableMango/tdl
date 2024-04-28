package uml

import (
	"context"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1/tdlv1alpha1connect"
)

type broker struct {
	client tdlv1alpha1connect.UmlServiceClient
}

type brokerOptions struct {
	baseUrl string
}

type BrokerOption = func(*brokerOptions)

type Broker interface {
	From(context.Context, io.Reader) (*Spec, error)
	To(context.Context, io.Writer, *Spec) error
}

func WithUrl(baseUrl string) BrokerOption {
	return func(bo *brokerOptions) {
		bo.baseUrl = baseUrl
	}
}

func NewBroker(opts ...BrokerOption) (Broker, error) {
	options := &brokerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	client := tdlv1alpha1connect.NewUmlServiceClient(
		http.DefaultClient,
		options.baseUrl,
		connect.WithGRPC(),
	)

	return &broker{client: client}, nil
}
