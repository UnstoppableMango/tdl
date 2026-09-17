package emit_test

import (
	"slices"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func session(b *irtest.Builder) *emit.Session {
	return emit.NewSession(&plugin.Request{Target: "x", Model: b.Model}, "X")
}

func TestWords(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want []string
	}{
		{"userID", []string{"user", "ID"}},
		{"HTTPServer", []string{"HTTP", "Server"}},
		{"last4", []string{"last4"}},
		{"sha256Hash", []string{"sha256", "Hash"}},
		{"line_item", []string{"line", "item"}},
		{"Draft", []string{"Draft"}},
		{"", nil},
	} {
		if got := emit.Words(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("Words(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCases(t *testing.T) {
	for _, tt := range []struct {
		fn       func(string) string
		name     string
		in, want string
	}{
		{emit.Pascal, "Pascal", "user_id", "UserId"},
		{emit.Pascal, "Pascal", "userID", "UserID"},
		{emit.Camel, "Camel", "UserID", "userID"},
		{emit.Camel, "Camel", "placed_at", "placedAt"},
		{emit.Snake, "Snake", "placedAt", "placed_at"},
		{emit.Snake, "Snake", "HTTPServer", "http_server"},
		{emit.ScreamingSnake, "ScreamingSnake", "inProgress", "IN_PROGRESS"},
		{emit.LastSegment, "LastSegment", "shop.billing.Order", "Order"},
	} {
		if got := tt.fn(tt.in); got != tt.want {
			t.Errorf("%s(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestOwnSkipsThePrelude(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Note"}, Node: &ir.Decl_Structure{Structure: &ir.Struct{}}})

	own := session(b).Own()
	if len(own) != 1 || own[0].GetMeta().GetName() != "Note" {
		t.Errorf("own = %v", own)
	}
}

// A model carries directives for every target block, so a lookup that did
// not filter would read another backend's.
func TestFindFiltersByTarget(t *testing.T) {
	s := session(irtest.New("shop"))
	all := []*ir.Directive{
		{Name: "name", Target: "y", Args: []*ir.Literal{irtest.Text("theirs")}},
		{Name: "name", Target: "x"},
		{Name: "name", Target: "x", Args: []*ir.Literal{irtest.Text("mine")}},
	}
	if got, ok := s.Text(all, "name"); !ok || got != "mine" {
		t.Errorf("Text = %q, %v", got, ok)
	}
}

func TestResolve(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Sku"}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: b.Named("string")}}})
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Tags"}, Node: &ir.Decl_Alias{Alias: &ir.Alias{Target: b.Named("Set", b.Named("string"))}}})

	s := session(b)
	resolve := func(id *ir.ID) *emit.Ref {
		t.Helper()
		r, err := s.Resolve(id)
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		return r
	}

	if r := resolve(b.Named("int")); r.Form != emit.Prim || r.Name != "int" {
		t.Errorf("int = %+v", r)
	}
	if r := resolve(b.Named("Tags")); r.Form != emit.Set || r.Elem.Name != "string" {
		t.Errorf("an alias was not expanded: %+v", r)
	}
	r := resolve(b.Named("Map", b.Named("Sku"), b.Named("Option", b.Named("List", b.Named("bool")))))
	if r.Form != emit.Map || r.Key.Form != emit.Named || r.Key.Decl.GetMeta().GetName() != "Sku" {
		t.Errorf("map key = %+v", r.Key)
	}
	if r.Elem.Form != emit.Option || r.Elem.Elem.Form != emit.List || r.Elem.Elem.Elem.Name != "bool" {
		t.Errorf("map value = %+v", r.Elem)
	}
	if r := resolve(b.Named("Nullable", b.Named("uuid"))); r.Form != emit.Nullable {
		t.Errorf("nullable = %+v", r)
	}
}

func TestResolveRefusesWithAPosition(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Auditable"}, Node: &ir.Decl_Class{Class: &ir.Class{}}})
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Page"}, Node: &ir.Decl_Structure{Structure: &ir.Struct{
		Params: []*ir.Param{{Name: "T"}},
	}}})
	at := &ir.Position{Filename: irtest.OwnFile, Line: 3}
	intern := func(t *ir.Type) *ir.ID {
		t.Position = at
		b.Model.Types = append(b.Model.Types, t)
		return &ir.ID{Index: int32(len(b.Model.Types) - 1)}
	}

	for name, id := range map[string]*ir.ID{
		"param":   intern(&ir.Type{Param: &ir.ParamRef{Name: "T"}}),
		"extern":  intern(&ir.Type{Extern: &ir.ID{Name: "common.Address"}}),
		"class":   intern(&ir.Type{Ctor: func() *ir.ID { _, id, _ := b.Model.FindDecl("Auditable"); return id }()}),
		"generic": intern(&ir.Type{Ctor: func() *ir.ID { _, id, _ := b.Model.FindDecl("Page"); return id }(), Args: []*ir.ID{b.Named("int")}}),
	} {
		_, err := session(b).Resolve(id)
		u, ok := err.(*emit.UnsupportedError)
		if !ok {
			t.Errorf("%s: err = %v", name, err)
			continue
		}
		if u.Position.GetLine() != 3 {
			t.Errorf("%s: position = %+v", name, u.Position)
		}
	}
}

// A declaration naming a skipped one would name something the output does
// not declare, so it is skipped too, through a collection and an alias.
func TestCascade(t *testing.T) {
	b := irtest.New("shop")
	structure := func(name string, fields ...*ir.Field) *ir.Decl {
		d := &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: fields}}}
		b.Own(d)
		return d
	}
	broken := structure("Broken")
	viaList := structure("ViaList", irtest.Field("all", b.Named("List", b.Named("Broken"))))
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Alias"}, Node: &ir.Decl_Alias{Alias: &ir.Alias{Target: b.Named("ViaList")}}})
	viaAlias := structure("ViaAlias", irtest.Field("a", b.Named("Alias")))
	fine := structure("Fine", irtest.Field("n", b.Named("int")))

	s := session(b)
	skipped := map[*ir.Decl]bool{broken: true}
	s.Cascade(s.Own(), skipped)

	if !skipped[viaList] || !skipped[viaAlias] {
		t.Errorf("referrers were not skipped: %v", skipped)
	}
	if skipped[fine] {
		t.Error("a declaration naming nothing skipped was skipped")
	}
	if len(s.Diags) != 2 {
		t.Errorf("diagnostics = %+v", s.Diags)
	}
}
