package spec

import (
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

func PipeFs[
	O c.Pipeline[afero.Fs, T],
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I, path string) O {
	return mediatype.PipeFs[O](pipeline, path, Zero)
}

func PipeInput[
	O c.Pipeline[tdl.Input, T],
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I) O {
	return mediatype.PipeInput[O](pipeline, Zero)
}

func PipeRead[
	O c.Pipeline[io.Reader, T],
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I, media tdl.MediaType) O {
	return mediatype.PipeRead[O](pipeline, media, Zero)
}
