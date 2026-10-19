package emit_test

import (
	"slices"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func lossSession() *emit.Session {
	return emit.NewSession(&plugin.Request{Target: "x", Model: &ir.Model{}}, "X")
}

func codes(s *emit.Session) []string {
	var out []string
	for _, d := range s.Diags {
		out = append(out, d.GetCode())
	}
	return out
}

func TestLoseKind(t *testing.T) {
	for _, tt := range []struct {
		kind ir.StructKind
		want string
	}{
		{ir.StructKind_STRUCT_KIND_ENTITY, "entity"},
		{ir.StructKind_STRUCT_KIND_MIXIN, "mixin"},
		{ir.StructKind_STRUCT_KIND_VALUE, ""},
	} {
		s := lossSession()
		d := &ir.Decl{Meta: &ir.Meta{Name: "T"}, Node: &ir.Decl_Structure{Structure: &ir.Struct{Kind: tt.kind}}}
		if got := s.LoseKind(d); got != tt.want {
			t.Errorf("LoseKind(%v) = %q, want %q", tt.kind, got, tt.want)
		}
		if lost := len(s.Diags) > 0; lost != (tt.want != "") {
			t.Errorf("LoseKind(%v) warned %v", tt.kind, codes(s))
		}
	}
}

func TestLoseConforms(t *testing.T) {
	entity := &ir.ClassRef{Class: &ir.ID{Name: "Entity"}}
	audited := &ir.ClassRef{Class: &ir.ID{Name: "Audited"}}
	for _, tt := range []struct {
		name string
		d    *ir.Decl
		want int
	}{
		{"struct's Entity", &ir.Decl{Node: &ir.Decl_Structure{Structure: &ir.Struct{Conforms: []*ir.ClassRef{entity}}}}, 0},
		{"struct's class", &ir.Decl{Node: &ir.Decl_Structure{Structure: &ir.Struct{Conforms: []*ir.ClassRef{entity, audited}}}}, 1},
		{"enum's Entity", &ir.Decl{Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{Conforms: []*ir.ClassRef{entity}}}}, 1},
		{"none", &ir.Decl{Node: &ir.Decl_Structure{Structure: &ir.Struct{}}}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := lossSession()
			if got := s.LoseConforms(tt.d, "X"); got != (tt.want > 0) {
				t.Errorf("LoseConforms = %v, want %v", got, tt.want > 0)
			}
			if len(s.Diags) != tt.want {
				t.Errorf("LoseConforms warned %d times, want %d", len(s.Diags), tt.want)
			}
		})
	}
}

func TestDeclaresLossy(t *testing.T) {
	meta := &ir.Meta{Name: "T"}
	for _, tt := range []struct {
		name     string
		d        *ir.Decl
		want     bool
		wantCode string
		aliased  bool
	}{
		{"struct", &ir.Decl{Meta: meta, Node: &ir.Decl_Structure{Structure: &ir.Struct{}}}, true, "", false},
		{"primitive", &ir.Decl{Meta: meta, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}}, false, "", false},
		{"class", &ir.Decl{Meta: meta, Node: &ir.Decl_Class{Class: &ir.Class{}}}, false, emit.LossClass, false},
		{"unit", &ir.Decl{Meta: meta, Node: &ir.Decl_Unit{Unit: &ir.UnitDef{}}}, false, emit.LossUnit, false},
		{"generic", &ir.Decl{Meta: meta, Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: []*ir.Param{{Name: "a"}}}}}, false, emit.LossGeneric, false},
		{"alias", &ir.Decl{Meta: meta, Node: &ir.Decl_Alias{Alias: &ir.Alias{Target: &ir.ID{Name: "string"}}}}, false, "", true},
		{"generic alias", &ir.Decl{Meta: meta, Node: &ir.Decl_Alias{Alias: &ir.Alias{Params: []*ir.Param{{Name: "a"}}}}}, false, "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := lossSession()
			aliased := false
			got, err := s.DeclaresLossy(tt.d, func(*ir.ID) { aliased = true })
			if got != tt.want {
				t.Errorf("DeclaresLossy = %v, want %v", got, tt.want)
			}
			var code string
			if u, ok := err.(*emit.UnsupportedError); ok {
				code = u.Code
			} else if err != nil {
				t.Fatalf("DeclaresLossy error = %v", err)
			}
			if code != tt.wantCode {
				t.Errorf("DeclaresLossy code = %q, want %q", code, tt.wantCode)
			}
			if aliased != tt.aliased {
				t.Errorf("aliased called = %v, want %v", aliased, tt.aliased)
			}
			if tt.aliased && !slices.Equal(codes(s), []string{emit.LossAlias}) {
				t.Errorf("DeclaresLossy warned %v, want [%s]", codes(s), emit.LossAlias)
			}
		})
	}
}

func TestLoseDeclName(t *testing.T) {
	named := []*ir.Directive{{Name: "name", Target: "x", Args: []*ir.Literal{irtest.Text("Order")}}}
	for _, tt := range []struct {
		name, decl, written string
		dirs                []*ir.Directive
		want                bool
	}{
		{"same", "Order", "Order", nil, false},
		{"written another way", "Order", "order", nil, true},
		{"restated directive", "Order", "Order", named, true},
		{"reader writes a directive", "order_line", "order_line", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := lossSession()
			d := &ir.Decl{Meta: &ir.Meta{Name: tt.decl}, Directives: tt.dirs}
			if got := s.LoseDeclName(d, tt.written); got != tt.want {
				t.Errorf("LoseDeclName = %v, want %v", got, tt.want)
			}
			if lost := len(s.Diags) > 0; lost != tt.want {
				t.Errorf("LoseDeclName warned %v", codes(s))
			}
		})
	}
}

func TestLoseFieldName(t *testing.T) {
	named := []*ir.Directive{{Name: "name", Target: "x", Args: []*ir.Literal{irtest.Text("id")}}}
	for _, tt := range []struct {
		name, written string
		dirs          []*ir.Directive
		want          bool
	}{
		{"same", "id", nil, false},
		{"written another way", "ID", nil, true},
		{"restated directive", "id", named, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := lossSession()
			f := &ir.Field{Meta: &ir.Meta{Name: "id"}, Directives: tt.dirs}
			if got := s.LoseFieldName("T", f, tt.written); got != tt.want {
				t.Errorf("LoseFieldName = %v, want %v", got, tt.want)
			}
			if lost := len(s.Diags) > 0; lost != tt.want {
				t.Errorf("LoseFieldName warned %v", codes(s))
			}
		})
	}
}

func TestLoseOwnedDefault(t *testing.T) {
	for _, tt := range []struct {
		name string
		f    *ir.Field
		want []string
	}{
		{"plain", &ir.Field{}, nil},
		{"owned", &ir.Field{Owned: true}, []string{emit.LossOwned}},
		{"default", &ir.Field{DefaultValue: irtest.Text("a")}, []string{emit.LossDefault}},
		{"both", &ir.Field{Owned: true, DefaultValue: irtest.Text("a")}, []string{emit.LossOwned, emit.LossDefault}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := lossSession()
			tt.f.Meta = &ir.Meta{Name: "f"}
			if got := s.LoseOwnedDefault("T", "X", tt.f); got != (len(tt.want) > 0) {
				t.Errorf("LoseOwnedDefault = %v, want %v", got, len(tt.want) > 0)
			}
			if got := codes(s); !slices.Equal(got, tt.want) {
				t.Errorf("LoseOwnedDefault warned %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExternOf(t *testing.T) {
	ext := &ir.Extern{Name: "Money"}
	s := emit.NewSession(&plugin.Request{Target: "x", Model: &ir.Model{Externs: []*ir.Extern{ext}}}, "X")
	for _, tt := range []struct {
		name string
		id   *ir.ID
		want *ir.Extern
	}{
		{"nil", nil, nil},
		{"in range", &ir.ID{Index: 0}, ext},
		{"past the end", &ir.ID{Index: 1}, nil},
		{"negative", &ir.ID{Index: -1}, nil},
	} {
		if got := s.ExternOf(tt.id); got != tt.want {
			t.Errorf("ExternOf(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
