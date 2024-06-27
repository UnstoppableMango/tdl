package gen

import (
	"context"
	"io"
	"testing"

	"github.com/unstoppablemango/tdl/pkg/uml"
)

type Test struct{}

//func TestGen(ctx context.Context, spec string, writer io.Writer) error {
//	return errors.New("TODO")
//}

func MapTest(t *testing.T) {
	gen := New(func(ctx context.Context, s *uml.Spec, w io.Writer) error {
		return nil
	})

	actual := MapI(gen, func(reader io.Reader) *uml.Spec {
		return nil
	})

	Run(context.TODO(), actual)
}
