package testing

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type MockGenerator struct {
	ExecuteFunc func(*tdlv1alpha1.Spec, tdl.Sink) error
}

// Execute implements tdl.Generator.
func (m *MockGenerator) Execute(spec *tdlv1alpha1.Spec, sink tdl.Sink) error {
	return m.ExecuteFunc(spec, sink)
}

var _ tdl.Generator = &MockGenerator{}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		ExecuteFunc: func(*tdlv1alpha1.Spec, tdl.Sink) error {
			panic("unimplemented")
		},
	}
}
