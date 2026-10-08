package emit_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

func TestDeclares(t *testing.T) {
	meta := &ir.Meta{Name: "x.T"}
	for _, tt := range []struct {
		name    string
		d       *ir.Decl
		want    bool
		wantErr string
	}{
		{"struct", &ir.Decl{Meta: meta, Node: &ir.Decl_Structure{Structure: &ir.Struct{}}}, true, ""},
		{"enum", &ir.Decl{Meta: meta, Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{}}}, true, ""},
		{"newtype", &ir.Decl{Meta: meta, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{}}}, true, ""},
		{"alias", &ir.Decl{Meta: meta, Node: &ir.Decl_Alias{Alias: &ir.Alias{}}}, false, ""},
		{"primitive", &ir.Decl{Meta: meta, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}}, false, ""},
		{"class", &ir.Decl{Meta: meta, Node: &ir.Decl_Class{Class: &ir.Class{}}}, false, "x.T is a class"},
		{"unit", &ir.Decl{Meta: meta, Node: &ir.Decl_Unit{Unit: &ir.UnitDef{}}}, false, "x.T is a unit"},
		{"generic", &ir.Decl{Meta: meta, Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: []*ir.Param{{Name: "a"}}}}}, false, "x.T is parameterized"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := emit.Declares(tt.d)
			if got != tt.want {
				t.Errorf("Declares = %v, want %v", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Declares error = %v, want none", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("Declares error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestDocComment(t *testing.T) {
	for _, tt := range []struct {
		name string
		meta *ir.Meta
		want string
	}{
		{"none", &ir.Meta{}, ""},
		{"doc", &ir.Meta{Doc: []string{"One.", "", "Two */ three."}}, "  /**\n   * One.\n   *\n   * Two * / three.\n   */\n"},
		{"deprecated", &ir.Meta{Deprecated: &ir.Deprecation{}}, "  /**\n   * @deprecated\n   */\n"},
		{"both", &ir.Meta{Doc: []string{"One."}, Deprecated: &ir.Deprecation{Reason: "use Two"}}, "  /**\n   * One.\n   *\n   * @deprecated use Two\n   */\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			emit.DocComment(&b, "  ", tt.meta)
			if got := b.String(); got != tt.want {
				t.Errorf("DocComment =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestConstraintText(t *testing.T) {
	lit := func(k ir.LiteralKind, text string) *ir.Literal { return &ir.Literal{Kind: k, Text: text} }
	n := func(v int64) *int64 { return &v }
	for _, tt := range []struct {
		c    *ir.Constraint
		want string
	}{
		{&ir.Constraint{Name: "unique"}, "unique"},
		{&ir.Constraint{Name: "max", Args: []*ir.Literal{lit(ir.LiteralKind_LITERAL_KIND_INT, "10")}}, "max(10)"},
		{&ir.Constraint{Name: "matches", Args: []*ir.Literal{lit(ir.LiteralKind_LITERAL_KIND_REGEX, "^a+$")}}, "matches(/^a+$/)"},
		{&ir.Constraint{Name: "oneOf", Args: []*ir.Literal{{
			Kind:  ir.LiteralKind_LITERAL_KIND_LIST,
			Items: []*ir.Literal{lit(ir.LiteralKind_LITERAL_KIND_STRING, `a"b`), lit(ir.LiteralKind_LITERAL_KIND_NAME, "Draft")},
		}}}, `oneOf(["a\"b", Draft])`},
		{&ir.Constraint{Name: "length", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{Low: n(1), High: n(5)}}}}, "length(1..5)"},
		{&ir.Constraint{Name: "length", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{Low: n(2)}}}}, "length(2..)"},
		{&ir.Constraint{Name: "length", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{High: n(3)}}}}, "length(..3)"},
	} {
		if got := emit.ConstraintText(tt.c); got != tt.want {
			t.Errorf("ConstraintText = %q, want %q", got, tt.want)
		}
	}
}
