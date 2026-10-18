package gen_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
)

var record = flag.Bool("record", false, "rewrite testdata/plugin exchanges")

// TestRecordedExchanges replays request and response pairs kept as
// protobuf text and binary, for an implementation in another language to
// replay. dotnet/Tdl.Plugin.Tests answers each binary request and compares
// its bytes with the binary response.
func TestRecordedExchanges(t *testing.T) {
	sources, err := filepath.Glob("../../testdata/plugin/*.tdl")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no exchanges found: %v", err)
	}

	for _, src := range sources {
		name := filepath.Base(src)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("reading %s: %v", src, err)
			}
			file, err := parser.Parse(name, strings.NewReader(string(data)))
			if err != nil {
				t.Fatalf("parsing: %v", err)
			}

			model, diags := sema.Lower(file)
			if len(diags) > 0 {
				t.Fatalf("lowering: %v", diags)
			}

			req, _ := analysis.Request(model, debug.Backend{}.Describe(), debug.Name, "gen")
			resp, err := debug.Backend{}.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			base := src[:len(src)-len(".tdl")]
			checkExchange(t, base+".request.txtpb", req)
			checkExchange(t, base+".response.txtpb", resp)
			checkBinary(t, base+".request.binpb", req)
			checkBinary(t, base+".response.binpb", resp)
		})
	}
}

func checkExchange(t *testing.T, path string, m proto.Message) {
	t.Helper()

	text, err := prototext.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(m)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if *record {
		if err := os.WriteFile(path, text, 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run `go test ./internal/gen -record`)", path, err)
	}

	// prototext output is unstable across builds, so compare parsed.
	golden := m.ProtoReflect().New().Interface()
	if err := prototext.Unmarshal(want, golden); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if !proto.Equal(m, golden) {
		t.Errorf("%s is out of date:\n--- got ---\n%s", path, text)
	}
}

// checkBinary is checkExchange for the wire encoding. Field order is
// deterministic without maps, which the protocol has none of, but the
// comparison is still parsed so a reordering is not a failure.
func checkBinary(t *testing.T, path string, m proto.Message) {
	t.Helper()

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if *record {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run `go test ./internal/gen -record`)", path, err)
	}
	golden := m.ProtoReflect().New().Interface()
	if err := proto.Unmarshal(want, golden); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if !proto.Equal(m, golden) {
		t.Errorf("%s is out of date (run `go test ./internal/gen -record`)", path)
	}
}
