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
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

var record = flag.Bool("record", false, "rewrite testdata/plugin exchanges")

// TestRecordedExchanges replays request and response pairs kept as
// protobuf text.
//
// They are the protocol written down in a form nothing in this repository
// has to interpret, so an implementation in another language can replay
// them and compare. A Go test asserting a Go backend against itself proves
// less than these do.
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

			req := &plugin.Request{Target: debug.Name, Model: model, Out: "gen"}
			resp, err := debug.Backend{}.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			base := src[:len(src)-len(".tdl")]
			checkExchange(t, base+".request.txtpb", req)
			checkExchange(t, base+".response.txtpb", resp)
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

	// prototext is deliberately unstable across builds, so the recorded
	// text is compared by parsing it back rather than byte for byte.
	golden := m.ProtoReflect().New().Interface()
	if err := prototext.Unmarshal(want, golden); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if !proto.Equal(m, golden) {
		t.Errorf("%s is out of date:\n--- got ---\n%s", path, text)
	}
}
