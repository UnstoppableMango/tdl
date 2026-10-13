package emit_test

import (
	"slices"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
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
