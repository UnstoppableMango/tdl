package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/parser"
)

// A reference through an import alias is checked against the declarations
// the dependency's parse tree holds, though the dependency is not lowered.
func TestAliasedReferenceIsChecked(t *testing.T) {
	deps := MapLoader{
		"dep/shipping.tdl": `package shop.shipping

type Address { line1: string }
type Stamped<T> { at: T }
type lower { x: string }
class Tracked { }
unit cm
primitive grams
`,
		"dep/broken.tdl": "package shop.broken\n\ntype {",
	}

	tests := []struct {
		name string
		src  string
		want string // empty when the file lowers cleanly
	}{
		{
			name: "a declared type",
			src: `
import "dep/shipping.tdl" as shipping
type Order { to: shipping.Address  at: shipping.Stamped<instant> }
`,
		},
		{
			name: "a declared class",
			src: `
import "dep/shipping.tdl" as shipping
type Order: shipping.Tracked { }
class Audited: shipping.Tracked { }
type Box<T> requires shipping.Tracked<T> { v: T }
`,
		},
		{
			name: "a primitive and a unit are exported whatever their case",
			src: `
import "dep/shipping.tdl" as shipping
type Parcel { weight: shipping.grams  width: decimal<shipping.cm> }
`,
		},
		{
			name: "an undeclared name",
			src: `
import "dep/shipping.tdl" as shipping
type Order { to: shipping.Nope }
`,
			want: `4:18: "dep/shipping.tdl" declares no Nope`,
		},
		{
			name: "an undeclared name in a type argument",
			src: `
import "dep/shipping.tdl" as shipping
type Order { to: [shipping.Nope] }
`,
			want: `"dep/shipping.tdl" declares no Nope`,
		},
		{
			name: "an undeclared class",
			src: `
import "dep/shipping.tdl" as shipping
type Order: shipping.Nope { }
`,
			want: `"dep/shipping.tdl" declares no Nope`,
		},
		{
			name: "an unexported name",
			src: `
import "dep/shipping.tdl" as shipping
type Order { to: shipping.lower }
`,
			want: "4:18: shipping.lower is not exported",
		},
		{
			name: "a class as a type",
			src: `
import "dep/shipping.tdl" as shipping
type Order { t: shipping.Tracked }
`,
			want: "4:17: shipping.Tracked is a class, not a type",
		},
		{
			name: "a type after requires",
			src: `
import "dep/shipping.tdl" as shipping
type Box<T> requires shipping.Address<T> { v: T }
`,
			want: "4:22: shipping.Address is not a class",
		},
		{
			name: "a type in a conformance list",
			src: `
import "dep/shipping.tdl" as shipping
type Order: shipping.Address { }
`,
			want: "shipping.Address is not a class",
		},
		{
			name: "a type as an instance's class",
			src: `
import "dep/shipping.tdl" as shipping
type Order { }
instance shipping.Address for Order
`,
			want: "shipping.Address is not a class",
		},
		{
			name: "a dependency that does not parse is reported once",
			src: `
import "dep/broken.tdl" as broken
type Order { to: broken.Anything }
`,
			want: `import "dep/broken.tdl" does not parse`,
		},
		{
			name: "a dependency that cannot be read is reported once",
			src: `
import "dep/missing.tdl" as missing
type Order { to: missing.Anything }
`,
			want: `cannot read import "dep/missing.tdl"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse("main.tdl", strings.NewReader("package shop.orders\n"+tt.src))
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			_, diags := Lower(file, WithLoader(deps))
			if tt.want == "" {
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return
			}
			if len(diags) != 1 || !strings.Contains(diags[0].Error(), tt.want) {
				t.Fatalf("diagnostics = %v, want one containing %q", diags, tt.want)
			}
		})
	}
}
