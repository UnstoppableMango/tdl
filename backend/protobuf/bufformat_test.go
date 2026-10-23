package protobuf_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// bufFormatModel exercises every place the backend writes an option or a
// body that may be empty.
const bufFormatModel = `package acme.widgets.v1

primitive Fn: type -> type -> type

type Empty {}

type Widget {
  name: string
  uid: string
  kind: Kind
}

enum Kind { Small Large }

enum Shape {
  Round
  Square { side: int32 }
}

type WidgetService {
  GetWidget: Fn<Empty, Widget>
  ListWidgets: Fn<Empty, Widget>
}

target protobuf for acme.widgets.v1 {
  import("google/api/field_behavior.proto")
  import("google/api/resource.proto")
  option("java_multiple_files", "true")
  Fn => rpc
  WidgetService => service
  Widget {
    option("(google.api.resource)", "{ type: \"acme/Widget\" pattern: \"widgets/{widget}\" singular: \"widget\" plural: \"widgets\" }")
    uid {
      option("(google.api.field_behavior)", "OUTPUT_ONLY")
      option("deprecated", "true")
    }
    kind => option("(google.api.field_behavior)", "OPTIONAL")
  }
  Kind.Large {
    option("deprecated", "true")
    option("debug_redact", "true")
  }
  WidgetService.GetWidget => option("deprecated", "true")
  WidgetService.ListWidgets {
    option("deprecated", "true")
    option("idempotency_level", "IDEMPOTENT")
  }
}
`

// The backend writes what buf format writes, so a project can hold its
// generated files to both buf format and tdl gen --verify.
func TestOutputIsBufFormatted(t *testing.T) {
	bin, err := exec.LookPath("buf")
	if err != nil {
		t.Skip("buf is not on PATH")
	}
	resp := generateModel(t, bufFormatModel)
	dir := t.TempDir()
	for _, f := range resp.GetFiles() {
		p := filepath.Join(dir, f.GetPath())
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, f.GetContent(), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(bin, "format", "-d", p).CombinedOutput()
		if err != nil || len(out) != 0 {
			t.Errorf("buf format -d %s: %v\n%s", f.GetPath(), err, out)
		}
	}
}
