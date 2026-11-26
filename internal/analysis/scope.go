package analysis

import (
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/unstoppablemango/tdl/ir"
)

var directiveName = (&ir.Directive{}).ProtoReflect().Descriptor().FullName()

// Scope returns a copy of m holding only target's directives and target
// blocks, which is what a request for target carries. m is left alone.
func Scope(m *ir.Model, target string) *ir.Model {
	out := proto.Clone(m).(*ir.Model)
	out.Targets = slices.DeleteFunc(out.Targets, func(b *ir.TargetBlock) bool {
		return b.GetMeta().GetName() != target
	})
	// A block's order is its index among the blocks the model holds.
	for i, b := range out.Targets {
		b.GetMeta().Order = int32(i)
	}
	scope(out.ProtoReflect(), target)
	return out
}

// scope drops every directive under msg that another target wrote.
func scope(msg protoreflect.Message, target string) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Message() == nil || fd.IsMap() {
			return true
		}
		if !fd.IsList() {
			scope(v.Message(), target)
			return true
		}
		list := v.List()
		if fd.Message().FullName() != directiveName {
			for i := range list.Len() {
				scope(list.Get(i).Message(), target)
			}
			return true
		}
		kept := 0
		for i := range list.Len() {
			if d := list.Get(i).Message().Interface().(*ir.Directive); d.GetTarget() == target {
				list.Set(kept, list.Get(i))
				kept++
			}
		}
		list.Truncate(kept)
		return true
	})
}
