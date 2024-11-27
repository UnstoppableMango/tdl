package testing

import (
	"context"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type MockGenerator struct {
	ExecuteFunc func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error)
	StringFunc  func() string
}

// String implements tdl.Generator.
func (m *MockGenerator) String() string {
	return m.StringFunc()
}

// Execute implements tdl.Generator.
func (m *MockGenerator) Execute(
	ctx context.Context,
	spec *tdlv1alpha1.Spec,
) (afero.Fs, error) {
	return m.ExecuteFunc(ctx, spec)
}

func (m *MockGenerator) WithExecute(
	fn func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error),
) *MockGenerator {
	m.ExecuteFunc = fn
	return m
}

func (m *MockGenerator) WithString(
	fn func() string,
) *MockGenerator {
	m.StringFunc = fn
	return m
}

func (m *MockGenerator) WithName(name string) *MockGenerator {
	return m.WithString(func() string {
		return name
	})
}

var _ tdl.Generator = &MockGenerator{}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		ExecuteFunc: func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error) {
			panic("unimplemented")
		},
	}
}
