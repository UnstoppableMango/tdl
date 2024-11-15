package testing

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type MockGenerator struct {
	ExecuteFunc func(*tdlv1alpha1.Spec, tdl.Sink) error
}

type MockGeneratorStringer struct {
	*MockGenerator
	StringFunc func() string
}

// Execute implements tdl.Generator.
func (m *MockGenerator) Execute(spec *tdlv1alpha1.Spec, sink tdl.Sink) error {
	return m.ExecuteFunc(spec, sink)
}

func (m *MockGenerator) WithExecute(
	fn func(*tdlv1alpha1.Spec, tdl.Sink) error,
) *MockGenerator {
	m.ExecuteFunc = fn
	return m
}

func (m *MockGenerator) WithString(
	fn func() string,
) *MockGeneratorStringer {
	return &MockGeneratorStringer{
		MockGenerator: m,
		StringFunc:    fn,
	}
}

func (m *MockGenerator) WithName(name string) *MockGeneratorStringer {
	return m.WithString(func() string {
		return name
	})
}

var _ tdl.SinkGenerator = &MockGenerator{}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		ExecuteFunc: func(*tdlv1alpha1.Spec, tdl.Sink) error {
			panic("unimplemented")
		},
	}
}
