package protobuf_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// googleAPI stands in for the two googleapis files the options come from,
// declaring only the extensions the tests use.
var googleAPI = map[string]string{
	"google/api/field_behavior.proto": `syntax = "proto3";
package google.api;
import "google/protobuf/descriptor.proto";
enum FieldBehavior {
  FIELD_BEHAVIOR_UNSPECIFIED = 0;
  OUTPUT_ONLY = 3;
}
extend google.protobuf.FieldOptions {
  repeated FieldBehavior field_behavior = 1052 [packed = false];
}
`,
	"google/api/field_info.proto": `syntax = "proto3";
package google.api;
import "google/protobuf/descriptor.proto";
message FieldInfo {
  enum Format {
    FORMAT_UNSPECIFIED = 0;
    UUID4 = 1;
  }
  Format format = 1;
}
extend google.protobuf.FieldOptions {
  FieldInfo field_info = 291403980;
}
`,
}

const fieldOptionsModel = `package shop

type Widget {
  id: string
  uid: uuid
}

target protobuf for shop {
  import("google/api/field_behavior.proto")
  import("google/api/field_info.proto")
  Widget {
    uid {
      number(2)
      option("(google.api.field_behavior)", "OUTPUT_ONLY")
      option("(google.api.field_info).format", "UUID4")
    }
  }
}
`

func TestOptionsOnAField(t *testing.T) {
	file, err := parser.Parse("shop.tdl", strings.NewReader(fieldOptionsModel))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v", diags)
	}

	if problems := gen.CheckDirectives(protobuf.Name, model, protobuf.Backend{}.Describe()); len(problems) != 0 {
		t.Errorf("problems = %v, want none: protobuf declares import and option, both repeatable", problems)
	}

	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{Target: protobuf.Name, Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, compileWith(t, resp, googleAPI),
		`import "google/api/field_behavior.proto";`,
		`import "google/api/field_info.proto";`,
		"string id = 1;",
		"string uid = 2 [(google.api.field_behavior) = OUTPUT_ONLY, (google.api.field_info).format = UUID4];",
	)
}

func option(name, value string) *ir.Directive {
	return &ir.Directive{Name: "option", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text(name), irtest.Text(value)}}
}

// An inlined oneof member stands for its variant's one field, so it carries
// that field's options and deprecation, and its variant's too.
func TestOptionsOnAnInlinedOneofMember(t *testing.T) {
	b := irtest.New("shop")
	contact := variant("Contact", irtest.Field("contact", b.Named("string")))
	contact.Directives = []*ir.Directive{option("(google.api.field_behavior)", "OUTPUT_ONLY")}
	system := variant("SystemActor", irtest.Field("system_actor", b.Named("string")))
	system.Meta.Deprecated = &ir.Deprecation{}
	fax := irtest.Field("fax", b.Named("string"))
	fax.Directives = []*ir.Directive{option("(google.api.field_info).format", "UUID4")}
	fax.Meta.Deprecated = &ir.Deprecation{}
	b.Own(enum("TriggerActor", contact, system, variant("Fax", fax)))

	actor := irtest.Field("actor", b.Named("TriggerActor"))
	actor.Directives = []*ir.Directive{{Name: "oneof", Target: protobuf.Name}}
	b.Own(value("Trigger", actor))
	b.Model.Targets = []*ir.TargetBlock{{
		Meta: &ir.Meta{Name: protobuf.Name},
		Directives: []*ir.Directive{
			{Name: "import", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("google/api/field_behavior.proto")}},
			{Name: "import", Target: protobuf.Name, Args: []*ir.Literal{irtest.Text("google/api/field_info.proto")}},
		},
	}}

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, compileWith(t, resp, googleAPI),
		`oneof actor {
  string contact = 1 [(google.api.field_behavior) = OUTPUT_ONLY];
  string system_actor = 2 [deprecated = true];
  string fax = 3 [deprecated = true, (google.api.field_info).format = UUID4];
}`,
	)
}
