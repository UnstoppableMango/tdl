package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/ir"
)

const (
	metaV1File = "k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto"
	metaV1Stub = `syntax = "proto2";
package k8s.io.apimachinery.pkg.apis.meta.v1;
message Condition { optional string type = 1; }
`
)

// foreign maps a declaration to a message another proto file declares, the
// way a target block's foreign directive does.
func foreign(d *ir.Decl, file, message string) *ir.Decl {
	d.Directives = append(d.Directives, &ir.Directive{
		Name:     "foreign",
		Target:   protobuf.Name,
		Args:     []*ir.Literal{irtest.Text(file), irtest.Text(message)},
		Position: &ir.Position{Filename: irtest.OwnFile, Line: 5},
	})
	return d
}

// A foreign declaration is a message the file imports rather than one it
// declares, so it generates an import and a fully qualified reference.
func TestForeignMessageIsImported(t *testing.T) {
	b := irtest.New("shop")
	b.Own(foreign(value("Condition", irtest.Field("type", b.Named("string"))),
		metaV1File, "k8s.io.apimachinery.pkg.apis.meta.v1.Condition"))
	b.Own(value("Widget", irtest.Field("conditions", b.Named("List", b.Named("Condition")))))

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := compileWith(t, resp, map[string]string{metaV1File: metaV1Stub})
	contains(t, src,
		`import "k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto";`,
		"message Widget { repeated k8s.io.apimachinery.pkg.apis.meta.v1.Condition conditions = 1; }",
	)
	absent(t, src, "message Condition")
}
