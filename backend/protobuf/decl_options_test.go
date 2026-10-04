package protobuf_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// googleAPIResource stands in for google/api/resource.proto.
var googleAPIResource = map[string]string{
	"google/api/resource.proto": `syntax = "proto3";
package google.api;
import "google/protobuf/descriptor.proto";
message ResourceDescriptor {
  string type = 1;
  repeated string pattern = 2;
  string plural = 5;
  string singular = 6;
}
extend google.protobuf.MessageOptions {
  ResourceDescriptor resource = 1053;
}
`,
}

// generateModel parses, lowers, and generates src, failing on any
// diagnostic.
func generateModel(t *testing.T, src string) *plugin.Response {
	t.Helper()
	file, err := parser.Parse("shop.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}
	if problems := gen.CheckDirectives(protobuf.Name, model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("problems = %v, want none", problems)
	}
	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	return resp
}

const messageOptionsModel = `package shop

type Account { id: string }

target protobuf for shop {
  import("google/api/resource.proto")
  Account {
    option("(google.api.resource)", "{ type: \"acme/Account\" pattern: \"accounts/{account}\" singular: \"account\" plural: \"accounts\" }")
    option("deprecated", "true")
  }
}
`

func TestOptionsOnAMessage(t *testing.T) {
	resp := generateModel(t, messageOptionsModel)
	contains(t, compileWith(t, resp, googleAPIResource),
		`import "google/api/resource.proto";`,
		`message Account {
  option (google.api.resource) = { type: "acme/Account" pattern: "accounts/{account}" singular: "account" plural: "accounts" };
  option deprecated = true;
  string id = 1;
}`,
	)
}

const enumOptionsModel = `package shop

enum State { Open Closed }

target protobuf for shop {
  State {
    option("deprecated", "true")
  }
}
`

func TestOptionsOnAnEnum(t *testing.T) {
	resp := generateModel(t, enumOptionsModel)
	contains(t, compile(t, resp),
		`enum State {
  option deprecated = true;
  STATE_UNSPECIFIED = 0;
  STATE_OPEN = 1;
  STATE_CLOSED = 2;
}`,
	)
}

const enumValueOptionsModel = `package shop

enum State { Open Closed }

target protobuf for shop {
  State {
    Closed {
      option("deprecated", "true")
    }
  }
}
`

func TestOptionsOnAnEnumValue(t *testing.T) {
	resp := generateModel(t, enumValueOptionsModel)
	contains(t, compile(t, resp),
		`enum State {
  STATE_UNSPECIFIED = 0;
  STATE_OPEN = 1;
  STATE_CLOSED = 2 [deprecated = true];
}`,
	)
}
