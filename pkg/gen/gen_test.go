package gen

import (
	"context"
	"errors"
	"io"
	"testing"
)

type Test struct{}

func TestGen(ctx context.Context, spec string, writer io.Writer) error {
	return errors.New("TODO")
}

func MapTest(t *testing.T) {
	gen := TestGen

	actual := MapI(gen, func(string) int {
		return 69
	})
}
