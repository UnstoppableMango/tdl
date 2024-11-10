package spec

import (
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

func PipeFs[
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I, path string) pipe.Func[afero.Fs, T] {
	return mediatype.PipeFs(pipeline, path, Zero)
}

func PipeInput[
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I) pipe.Func[tdl.Input, T] {
	return mediatype.PipeInput(pipeline, Zero)
}

func PipeRead[
	I c.Pipeline[*tdlv1alpha1.Spec, T],
	T any,
](pipeline I, media tdl.MediaType) pipe.Func[io.Reader, T] {
	return mediatype.PipeRead(pipeline, media, Zero)
}
