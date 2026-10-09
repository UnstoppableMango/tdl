package roundtrip_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/backend/graphql"
	"github.com/unstoppablemango/tdl/backend/internal/roundtrip"
	"github.com/unstoppablemango/tdl/backend/jsonschema"
	"github.com/unstoppablemango/tdl/backend/likec4"
	"github.com/unstoppablemango/tdl/backend/openapi"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/backend/thrift"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const (
	corpus = "../../../testdata/roundtrip"
	smoke  = "../../../testdata/gen/smoke/source.tdl"
)

// targets is every backend the round-trip goal covers, which is every one
// but debug, with its normal form once it has a reader.
var targets = []roundtrip.Target{
	{Backend: golang.Backend{}},
	{Backend: graphql.Backend{}},
	{Backend: jsonschema.Backend{}},
	{Backend: likec4.Backend{}},
	{Backend: openapi.Backend{}},
	{Backend: protobuf.Backend{}, Normalize: protobuf.Normalize},
	{Backend: salesforce.Backend{}},
	{Backend: smithy.Backend{}},
	{Backend: thrift.Backend{}},
	{Backend: typescript.Backend{}},
}

// TestCorpus runs every target's cases in testdata/roundtrip/<target>, and
// testdata/gen/smoke as a model-first case for each.
func TestCorpus(t *testing.T) {
	for _, target := range targets {
		name := target.Backend.Describe().Name
		t.Run(name, func(t *testing.T) {
			roundtrip.Run(t, target, filepath.Join(corpus, name), smoke)
		})
	}
}

// Every conformance case that imports nothing comes back from every target
// that imports, with annotations, the way smoke does. An import names a
// file a reverse backend is not given.
func TestConformanceComesBack(t *testing.T) {
	sources, err := filepath.Glob("../../../testdata/conformance/*/source.tdl")
	if err != nil {
		t.Fatal(err)
	}
	var standalone []string
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.Parse(path, strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		if len(file.Imports) == 0 {
			standalone = append(standalone, path)
		}
	}
	for _, target := range targets {
		if !target.Backend.Describe().Reverse {
			continue
		}
		t.Run(target.Backend.Describe().Name, func(t *testing.T) {
			for _, path := range standalone {
				t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
					roundtrip.Run(t, target, filepath.Join(t.TempDir(), "none"), path)
				})
			}
		})
	}
}

// Every shipped backend but debug is a target, and every corpus directory
// names one.
func TestEveryTargetIsCovered(t *testing.T) {
	var names []string
	for _, target := range targets {
		names = append(names, target.Backend.Describe().Name)
	}

	plugins, err := filepath.Glob("../../../cmd/tdl-gen-*")
	if err != nil {
		t.Fatal(err)
	}
	var shipped []string
	for _, p := range plugins {
		if name := strings.TrimPrefix(filepath.Base(p), "tdl-gen-"); name != "debug" {
			shipped = append(shipped, name)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, shipped) {
		t.Errorf("targets %v, shipped %v", names, shipped)
	}

	dirs, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if d.IsDir() && !slices.Contains(names, d.Name()) {
			t.Errorf("testdata/roundtrip/%s names no target", d.Name())
		}
	}
}

// The harness runs both directions against the echo backend, which imports
// exactly what it generated and warns lossy.order on every import.
func TestHarness(t *testing.T) {
	const src = `package shop

/// An order.
type Order: Entity {
  id: string
  tags: {string}
}

target echo for shop {
  out("gen")
}
`
	dir := t.TempDir()
	write(t, filepath.Join(dir, "model", roundtrip.Source), src)

	// Schema first: the JSON echo writes for src, which imports to src.
	write(t, filepath.Join(dir, "schema", roundtrip.Expected), src)
	write(t, filepath.Join(dir, "schema", roundtrip.Golden), "lossy.order\n")
	model := lowerFile(t, filepath.Join(dir, "schema", roundtrip.Expected), src)
	data, err := protojson.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "schema", echo.File), string(data))

	roundtrip.Run(t, roundtrip.Target{Backend: echo.Backend{}, Normalize: normalizeModel},
		dir, filepath.Join(dir, "model", roundtrip.Source))
}

// normalizeModel is echo's normal form: the model without positions, in
// deterministic binary.
func normalizeModel(files []*plugin.File) ([]*plugin.File, error) {
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		var m ir.Model
		if err := protojson.Unmarshal(f.GetContent(), &m); err != nil {
			return nil, err
		}
		data, err := proto.MarshalOptions{Deterministic: true}.Marshal(ir.WithoutPositions(&m))
		if err != nil {
			return nil, err
		}
		out[i] = &plugin.File{Path: f.GetPath(), Content: data}
	}
	return out, nil
}

func lowerFile(t *testing.T, path, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	return model
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
