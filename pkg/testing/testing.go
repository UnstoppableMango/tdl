package testing

import (
	"github.com/spf13/afero"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Test struct {
	Name     string
	Spec     *tdlv1alpha1.Spec
	Expected afero.Fs
}

type RawTest struct {
	Name   string
	Input  []byte
	Output []byte
}
