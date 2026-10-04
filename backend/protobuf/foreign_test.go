package protobuf_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

const (
	metaV1File = "k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto"
	metaV1Stub = `syntax = "proto2";
package k8s.io.apimachinery.pkg.apis.meta.v1;
message Condition { optional string type = 1; }
`
)

// foreign maps a declaration to a message another proto file declares.
func foreign(d *ir.Decl, file, message string) *ir.Decl {
	d.Directives = append(d.Directives, &ir.Directive{
		Name:     "foreign",
		Target:   protobuf.Name,
		Args:     []*ir.Literal{irtest.Text(file), irtest.Text(message)},
		Position: &ir.Position{Filename: irtest.OwnFile, Line: 5},
	})
	return d
}

// A foreign declaration generates an import and a fully qualified
// reference.
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

// foreignDirective is the directive [foreign] attaches, for an extern.
func foreignDirective(file, message string) *ir.Directive {
	return foreign(&ir.Decl{}, file, message).GetDirectives()[0]
}

func TestForeignExternIsImported(t *testing.T) {
	b := irtest.New("shop")
	condition := b.ExternIn("k8s.io.apimachinery.pkg.apis.meta.v1", "Condition",
		foreignDirective(metaV1File, "k8s.io.apimachinery.pkg.apis.meta.v1.Condition"))
	b.Own(value("Widget", irtest.Field("conditions", b.Named("List", condition))))

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := compileWith(t, resp, map[string]string{metaV1File: metaV1Stub})
	contains(t, src,
		`import "k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto";`,
		"message Widget { repeated k8s.io.apimachinery.pkg.apis.meta.v1.Condition conditions = 1; }",
	)
}

// The warning names the extern.
func TestUnmappedExternIsSkipped(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))
	b.Own(value("Widget", irtest.Field("conditions",
		b.Named("List", b.ExternIn("k8s.io.apimachinery.pkg.apis.meta.v1", "Condition")))))

	resp := generate(t, b)
	diags := resp.GetDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", diags)
	}
	if diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", diags[0].GetSeverity())
	}
	contains(t, diags[0].GetMessage(), "k8s.io.apimachinery.pkg.apis.meta.v1.Condition")
	src := compile(t, resp)
	absent(t, src, "message Widget")
	contains(t, src, "message Fine")
}

// newtype declares an owned newtype over base.
func newtype(name string, base *ir.ID) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base}}}
}

// at gives an interned type reference a source position.
func at(b *irtest.Builder, id *ir.ID, line int32) *ir.ID {
	b.Model.GetTypes()[id.GetIndex()].Position = &ir.Position{Filename: irtest.OwnFile, Line: line}
	return id
}

// One warning names the extern, placed where the newtype names it.
func TestNewtypeOverUnmappedExternIsSkipped(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("Cond", at(b, b.ExternIn("k8s.io.apimachinery.pkg.apis.meta.v1", "Condition"), 3)))
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))
	b.Own(value("Widget", irtest.Field("condition", at(b, b.Named("Cond"), 9))))

	resp := generate(t, b)
	diags := resp.GetDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", diags)
	}
	if diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", diags[0].GetSeverity())
	}
	if pos := diags[0].GetPosition(); pos.GetFilename() != irtest.OwnFile || pos.GetLine() != 3 {
		t.Errorf("position = %+v, want %s:3, where the extern is named", pos, irtest.OwnFile)
	}
	contains(t, diags[0].GetMessage(), "k8s.io.apimachinery.pkg.apis.meta.v1.Condition")
	src := compile(t, resp)
	absent(t, src, "message Widget", "message Cond")
	contains(t, src, "message Fine { string a = 1; }")
}

func TestNewtypeOverForeignExternIsImported(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("Cond", b.ExternIn("k8s.io.apimachinery.pkg.apis.meta.v1", "Condition",
		foreignDirective(metaV1File, "k8s.io.apimachinery.pkg.apis.meta.v1.Condition"))))
	b.Own(value("Widget", irtest.Field("condition", b.Named("Cond"))))

	resp := generate(t, b)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := compileWith(t, resp, map[string]string{metaV1File: metaV1Stub})
	contains(t, src,
		`import "k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto";`,
		"message Widget { k8s.io.apimachinery.pkg.apis.meta.v1.Condition condition = 1; }",
	)
	absent(t, src, "message Cond")
}
