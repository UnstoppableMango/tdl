package ir

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// WithoutPositions returns a copy of m with every [Position] cleared, so two
// models of the same declarations compare equal with [proto.Equal] wherever
// the source was written.
func WithoutPositions(m *Model) *Model {
	out := proto.Clone(m).(*Model)
	clearPositions(out.ProtoReflect())
	return out
}

func clearPositions(m protoreflect.Message) {
	var clear []protoreflect.FieldDescriptor
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		md := fd.Message()
		switch {
		case md == nil:
		case md == (&Position{}).ProtoReflect().Descriptor():
			clear = append(clear, fd)
		case fd.IsList():
			list := v.List()
			for i := range list.Len() {
				clearPositions(list.Get(i).Message())
			}
		case !fd.IsMap():
			clearPositions(v.Message())
		}
		return true
	})
	for _, fd := range clear {
		m.Clear(fd)
	}
}
