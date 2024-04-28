package broker

import (
	"net/http"

	"connectrpc.com/connect"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1/tdlv1alpha1connect"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type broker struct {
	client tdl.UmlServiceClient
}

type brokerOptions struct {
	baseUrl string
}

type BrokerOption = func(*brokerOptions)

type Broker interface {
	uml.Converter
	uml.Generator
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

	client := tdl.NewUmlServiceClient(
		http.DefaultClient,
		options.baseUrl,
		connect.WithGRPC(),
	)

	return &broker{client: client}, nil
}
