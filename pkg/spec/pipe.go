package spec

import (
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

func PipeRead[
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	O c.Pipeline[io.Reader, T],
	T any,
](pipeline I, media tdl.MediaType) O {
	return mediatype.PipeRead[I, O](pipeline, media, Zero)
}