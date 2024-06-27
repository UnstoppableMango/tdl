package gen

import (
	"io"

	"github.com/unstoppablemango/tdl/pkg/result"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func FromMediaType(g GeneratorFunc[*uml.Spec, io.Writer], mediaType string) GeneratorFunc[io.Reader, io.Writer] {
	return MapI(g, func(reader io.Reader) result.R[*uml.Spec] {
		data, err := io.ReadAll(reader)
		if err != nil {
			return result.OfErr[*uml.Spec](err)
		}

		spec := &uml.Spec{}
		if err = uml.Unmarshal(mediaType, data, spec); err != nil {
			return result.OfErr[*uml.Spec](err)
		}

		return result.Of(spec)
	})
}
