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

// foreignDirective is the directive [foreign] attaches, for a declaration
// the model holds only as an extern.
func foreignDirective(file, message string) *ir.Directive {
	return foreign(&ir.Decl{}, file, message).GetDirectives()[0]
}

// A target path can name a declaration another package owns, and a foreign
// mapping on it is read the way one on a local declaration is.
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

// An extern nothing maps has no protobuf message, so the declaration using
// it is skipped with a warning naming the extern.
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

// newtype declares a newtype over base, owned by the model.
func newtype(name string, base *ir.ID) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base}}}
}

// at gives an interned type reference the position lowering records for
// where it was written.
func at(b *irtest.Builder, id *ir.ID, line int32) *ir.ID {
	b.Model.GetTypes()[id.GetIndex()].Position = &ir.Position{Filename: irtest.OwnFile, Line: line}
	return id
}

// A newtype over an extern nothing maps expands to nothing protobuf has, so
// the one message using it is skipped with one warning naming the extern,
// placed where the newtype names the extern.
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

// A newtype over a mapped extern expands to the foreign message, the way a
// field naming the extern directly does.
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
