package analysis_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/plugin"
)

const carried = `package shop

/// An order.
type Order: Entity {
  id: string
  deprecated("use ref") code: string where { length(1..8) }
}

enum Status {
  Open
  Closed { at: string }
}

target wire for shop {
  roundtrip
  Order => key(id)
}
`

func TestCarry(t *testing.T) {
	req, diags := analysis.Request(lower(t, carried), plugin.Description{Name: "wire"}, "wire", "out")
	if len(diags) > 0 {
		t.Fatalf("diagnostics = %v", diags)
	}
	src := req.GetAnalysis().GetSource()
	items := src.GetItems()
	if len(items) != 3 {
		t.Fatalf("items = %v", items)
	}
	order := "/// An order.\ntype Order: Entity {\n  id: string\n  deprecated(\"use ref\") code: string where { length(1..8) }\n}"
	if items[0].GetName() != "Order" || items[0].GetSource() != order || items[0].GetConforms() != "Entity" {
		t.Errorf("Order = %v", items[0])
	}
	if items[1].GetName() != "Status" || items[1].GetConforms() != "" {
		t.Errorf("Status = %v", items[1])
	}
	if block := "target wire for shop {\n  Order => key(id)\n}"; items[2].GetName() != "" || items[2].GetSource() != block {
		t.Errorf("the target block = %v, want %q", items[2], block)
	}
	if len(src.GetImports()) != 0 {
		t.Errorf("imports = %v", src.GetImports())
	}

	fields := view(t, req.GetModel(), req.GetAnalysis(), "Order").GetFields()
	if fields[0].GetSource() != "id: string" || fields[0].GetDeprecation() != "" {
		t.Errorf("id = %v", fields[0])
	}
	if fields[1].GetSource() != "code: string where { length(1..8) }" || fields[1].GetDeprecation() != `deprecated("use ref")` {
		t.Errorf("code = %v", fields[1])
	}
	at := view(t, req.GetModel(), req.GetAnalysis(), "Status").GetVariants()[1].GetFields()[0]
	if at.GetSource() != "at: string" {
		t.Errorf("Closed.at = %v", at)
	}
}

// A target block holding only roundtrip is carried as nothing, and another
// target's request carries no source.
func TestCarryOnlyUnderRoundtrip(t *testing.T) {
	m := lower(t, "package shop\n\ntype Order {\n  id: string\n}\n\ntarget wire for shop {\n  roundtrip\n}\n")
	req, _ := analysis.Request(m, plugin.Description{Name: "wire"}, "wire", "out")
	items := req.GetAnalysis().GetSource().GetItems()
	if len(items) != 2 || items[1].GetSource() != "" {
		t.Errorf("items = %v", items)
	}

	req, _ = analysis.Request(m, plugin.Description{Name: "other"}, "other", "out")
	if src := req.GetAnalysis().GetSource(); src != nil {
		t.Errorf("source without roundtrip = %v", src)
	}
	if f := view(t, req.GetModel(), req.GetAnalysis(), "Order").GetFields()[0]; f.GetSource() != "" {
		t.Errorf("field source without roundtrip = %q", f.GetSource())
	}
}

// Only a bare roundtrip in the block itself asks for the source: another
// bare directive does not, and roundtrip on a declaration stays in the
// carried block.
func TestRoundtripIsTheBlocksBareDirective(t *testing.T) {
	m := lower(t, "package shop\n\ntype Order {\n  id: string\n}\n\ntarget other for shop {\n  flag\n}\n")
	if analysis.Roundtrip(m, "other") {
		t.Error("a bare flag is roundtrip")
	}

	m = lower(t, "package shop\n\ntype Order {\n  id: string\n}\n\ntarget wire for shop {\n  roundtrip\n  Order => roundtrip\n}\n")
	items := analysis.ModelSource(m, "wire").GetItems()
	if block := "target wire for shop {\n  Order => roundtrip\n}"; len(items) != 2 || items[1].GetSource() != block {
		t.Errorf("items = %v, want the block %q", items, block)
	}

	if analysis.WithoutRoundtrip(&ast.TargetDecl{N: "wire"}) == nil {
		t.Error("an empty block is dropped")
	}
}

func TestImported(t *testing.T) {
	m, err := analysis.Imported(&plugin.ImportResponse{Source: "package shop\n\ntype Order {\n  id: string\n}\n"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := m.FindDecl("Order"); !ok {
		t.Errorf("no Order in %v", m)
	}

	if _, err := analysis.Imported(&plugin.ImportResponse{Source: "type Order { id: Missing }\n"}); err == nil {
		t.Error("a source that does not lower imports")
	}
	if m, err := analysis.Imported(&plugin.ImportResponse{}); m != nil || err != nil {
		t.Errorf("an empty response = %v, %v", m, err)
	}
}
