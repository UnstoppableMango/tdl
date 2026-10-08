package protobuf

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/bufbuild/protocompile/experimental/fdp"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
	protoir "github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/unstoppablemango/tdl/plugin"
)

// compiled is .proto files compiled together, every custom option decoded
// as a field of its options message.
type compiled struct {
	// files is every file, imports included, each import before its
	// importer.
	files []*descriptorpb.FileDescriptorProto
	types *protoregistry.Types
}

// compileError is what protocompile reported about files that do not
// compile.
type compileError struct{ text string }

func (e *compileError) Error() string { return e.text }

// compile compiles the files at paths, importing from sources, the
// well-known types, and tdl/annotations.proto. It uses protocompile's
// experimental compiler, which handles edition 2024.
func compile(ctx context.Context, sources map[string]string, paths []string, sourceInfo bool) (*compiled, error) {
	files := source.NewMap(nil)
	for _, p := range slices.Sorted(maps.Keys(sources)) {
		files.Add(p, sources[p])
	}
	if _, ok := sources[annotationsFile]; !ok {
		files.Add(annotationsFile, string(annotationsProto))
	}
	results, diags, err := incremental.Run(ctx, incremental.New(), queries.FDS{
		Opener:    &source.Openers{files, source.WKTs()},
		Session:   new(protoir.Session),
		Workspace: source.NewWorkspace(paths...),
		Options:   *(&fdp.Options{}).Apply(fdp.IncludeSourceCodeInfo(sourceInfo)),
	})
	if err != nil {
		return nil, err
	}
	if text, errs, _ := (report.Renderer{}).RenderString(diags); errs != 0 {
		return nil, &compileError{text}
	}
	if fatal := results[0].Fatal; fatal != nil {
		return nil, &compileError{fatal.Error()}
	}

	set := results[0].Value
	registry, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, err
	}
	types := new(protoregistry.Types)
	var register func(xs protoreflect.ExtensionDescriptors, msgs protoreflect.MessageDescriptors)
	register = func(xs protoreflect.ExtensionDescriptors, msgs protoreflect.MessageDescriptors) {
		for i := range xs.Len() {
			if err == nil {
				err = types.RegisterExtension(dynamicpb.NewExtensionType(xs.Get(i)))
			}
		}
		for i := range msgs.Len() {
			register(msgs.Get(i).Extensions(), msgs.Get(i).Messages())
		}
	}
	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		register(fd.Extensions(), fd.Messages())
		return true
	})
	if err != nil {
		return nil, err
	}

	// Options arrive with custom options as unknown fields; reading them
	// again with every extension registered decodes them.
	out := &compiled{types: types}
	for _, f := range set.GetFile() {
		data, err := proto.Marshal(f)
		if err != nil {
			return nil, err
		}
		decoded := new(descriptorpb.FileDescriptorProto)
		if err := (proto.UnmarshalOptions{Resolver: types}).Unmarshal(data, decoded); err != nil {
			return nil, err
		}
		out.files = append(out.files, decoded)
	}
	return out, nil
}

// file is the compiled file at path, or nil.
func (c *compiled) file(path string) *descriptorpb.FileDescriptorProto {
	for _, f := range c.files {
		if f.GetName() == path {
			return f
		}
	}
	return nil
}

// Normalize puts .proto files into protobuf's normal form: each one's
// FileDescriptorProto without source info, as text, with its imports
// sorted and reserved ranges merged, since neither order means anything.
// Two schemas meaning the same compare equal in it, whatever their layout
// and comments.
func Normalize(files []*plugin.File) ([]*plugin.File, error) {
	sources := map[string]string{}
	var paths []string
	for _, f := range files {
		sources[f.GetPath()] = string(f.GetContent())
		paths = append(paths, f.GetPath())
	}
	c, err := compile(context.Background(), sources, paths, false)
	if err != nil {
		return nil, err
	}
	out := make([]*plugin.File, 0, len(files))
	for _, f := range files {
		fd := c.file(f.GetPath())
		if fd == nil {
			return nil, fmt.Errorf("%s did not compile", f.GetPath())
		}
		fd = proto.Clone(fd).(*descriptorpb.FileDescriptorProto)
		fd.SourceCodeInfo = nil
		if len(fd.GetPublicDependency()) == 0 && len(fd.GetWeakDependency()) == 0 {
			slices.Sort(fd.Dependency)
		}
		for _, m := range fd.GetMessageType() {
			normalMessage(m)
		}
		for _, e := range fd.GetEnumType() {
			e.ReservedRange = mergeEnumRanges(e.GetReservedRange())
			slices.Sort(e.ReservedName)
		}
		text, err := prototext.MarshalOptions{Multiline: true}.Marshal(fd)
		if err != nil {
			return nil, err
		}
		out = append(out, &plugin.File{Path: f.GetPath(), Content: text})
	}
	return out, nil
}

func normalMessage(m *descriptorpb.DescriptorProto) {
	m.ReservedRange = mergeRanges(m.GetReservedRange())
	slices.Sort(m.ReservedName)
	for _, n := range m.GetNestedType() {
		normalMessage(n)
	}
	for _, e := range m.GetEnumType() {
		e.ReservedRange = mergeEnumRanges(e.GetReservedRange())
		slices.Sort(e.ReservedName)
	}
}

// mergeRanges sorts a message's reserved ranges, whose ends are exclusive,
// and joins adjacent ones.
func mergeRanges(in []*descriptorpb.DescriptorProto_ReservedRange) []*descriptorpb.DescriptorProto_ReservedRange {
	sorted := slices.SortedFunc(slices.Values(in), func(a, b *descriptorpb.DescriptorProto_ReservedRange) int {
		return cmp.Compare(a.GetStart(), b.GetStart())
	})
	var out []*descriptorpb.DescriptorProto_ReservedRange
	for _, r := range sorted {
		if n := len(out); n > 0 && out[n-1].GetEnd() >= r.GetStart() {
			out[n-1].End = proto.Int32(max(out[n-1].GetEnd(), r.GetEnd()))
			continue
		}
		out = append(out, &descriptorpb.DescriptorProto_ReservedRange{Start: proto.Int32(r.GetStart()), End: proto.Int32(r.GetEnd())})
	}
	return out
}

// mergeEnumRanges is [mergeRanges] for an enum, whose ends are inclusive.
func mergeEnumRanges(in []*descriptorpb.EnumDescriptorProto_EnumReservedRange) []*descriptorpb.EnumDescriptorProto_EnumReservedRange {
	sorted := slices.SortedFunc(slices.Values(in), func(a, b *descriptorpb.EnumDescriptorProto_EnumReservedRange) int {
		return cmp.Compare(a.GetStart(), b.GetStart())
	})
	var out []*descriptorpb.EnumDescriptorProto_EnumReservedRange
	for _, r := range sorted {
		if n := len(out); n > 0 && out[n-1].GetEnd()+1 >= r.GetStart() {
			out[n-1].End = proto.Int32(max(out[n-1].GetEnd(), r.GetEnd()))
			continue
		}
		out = append(out, &descriptorpb.EnumDescriptorProto_EnumReservedRange{Start: proto.Int32(r.GetStart()), End: proto.Int32(r.GetEnd())})
	}
	return out
}
