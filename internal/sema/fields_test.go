package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/parser"
)

func TestClassFieldsAreRequired(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "declared conformance",
			src: `
class Auditable { createdAt: instant }
type Order: Auditable { id: string }
`,
			want: "Order has no field createdAt, required by Auditable",
		},
		{
			name: "a required class's field",
			src: `
class Timestamped { createdAt: instant }
class Auditable: Timestamped { updatedAt: instant }
type Order: Auditable { updatedAt: instant }
`,
			want: "Order has no field createdAt, required by Timestamped",
		},
		{
			name: "instance",
			src: `
class Auditable { createdAt: instant }
type Order { id: string }
instance Auditable for Order
`,
			want: "Order has no field createdAt, required by Auditable",
		},
		{
			name: "conditional instance",
			src: `
class Auditable { createdAt: instant }
type Page<T> { items: [T] }
instance <T> Auditable<Page<T>> requires Auditable<T>
`,
			want: "Page has no field createdAt, required by Auditable",
		},
		{
			name: "enum",
			src: `
class Auditable { createdAt: instant }
enum Status: Auditable { Draft }
`,
			want: "Status has no field createdAt, required by Auditable",
		},
		{
			name: "wrong type",
			src: `
class Auditable { createdAt: instant }
type Order: Auditable { createdAt: string }
`,
			want: "Order.createdAt is string, but Auditable requires instant",
		},
		{
			name: "optional is a different type",
			src: `
class Auditable { createdAt: instant }
type Order: Auditable { createdAt: instant? }
`,
			want: "Order.createdAt is Option<instant>, but Auditable requires instant",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := lowerDiags(t, tt.src)
			if !strings.Contains(diags.Error(), tt.want) {
				t.Errorf("diagnostics = %v, want %q", diags, tt.want)
			}
		})
	}
}

func TestClassFieldsAreSatisfied(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "through a mixin",
			src: `
class Auditable { createdAt: instant }
mixin Stamps { createdAt: instant }
type Order: Auditable { include Stamps }
`,
		},
		{
			name: "spelled differently",
			src: `
class Tagged { tags: List<string> }
type Order: Tagged { tags: [string] }
`,
		},
		{
			name: "through an alias",
			src: `
alias Tags = [string]
class Tagged { tags: Tags }
type Order: Tagged { tags: [string] }
`,
		},
		{
			name: "a field of a parameter's type",
			src: `
class Auditable { createdAt: instant }
type Stamped<T>: Auditable { createdAt: T }
`,
		},
		{
			name: "a conditional instance of a class with no fields",
			src: `
class Archived { }
type Page<T> { items: [T] }
instance <T> Archived<Page<T>> requires Archived<T>
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lower(t, tt.src)
		})
	}
}

func TestClassFieldsOfAnImportedType(t *testing.T) {
	deps := MapLoader{
		"dep/shipping.tdl": `package shop.shipping

unit cm
unit N = kg*m/s^2

type Address { line1: string  createdAt: instant  width: decimal<cm> }
type Parcel { line1: string }
type Stamp { createdAt: string }
type Box { width: decimal<cm> }
type Crate { width: decimal<m> }
type Push { force: decimal<kg*m/s^2> }
type Labelled { tags: [string]  owner: Address }
`,
	}

	tests := []struct {
		name string
		src  string
		want string // empty when the file lowers cleanly
	}{
		{
			name: "has the field",
			src: `
import "dep/shipping.tdl" as shipping
class Auditable { createdAt: instant }
instance Auditable for shipping.Address
`,
		},
		{
			name: "missing field",
			src: `
import "dep/shipping.tdl" as shipping
class Auditable { createdAt: instant }
instance Auditable<shipping.Parcel>
`,
			want: "shop.shipping.Parcel has no field createdAt, required by Auditable",
		},
		{
			name: "a required class's field, through a _ import",
			src: `
import "dep/shipping.tdl" as _
class Timestamped { createdAt: instant }
class Auditable: Timestamped { }
instance Auditable for Parcel
`,
			want: "shop.shipping.Parcel has no field createdAt, required by Timestamped",
		},
		{
			name: "wrong type",
			src: `
import "dep/shipping.tdl" as shipping
class Auditable { createdAt: instant }
instance Auditable for shipping.Stamp
`,
			want: "shop.shipping.Stamp.createdAt is string, but Auditable requires instant",
		},
		{
			name: "types naming the dependency's own declarations",
			src: `
import "dep/shipping.tdl" as shipping
class Tagged { tags: [string]  owner: shipping.Address }
instance Tagged for shipping.Labelled
`,
		},
		{
			name: "a unit the dependency declares",
			src: `
import "dep/shipping.tdl" as shipping
class Wide { width: decimal<shipping.cm> }
instance Wide for shipping.Box
`,
		},
		{
			name: "a derived unit, spelled two ways",
			src: `
import "dep/shipping.tdl" as shipping
class Pushed { force: decimal<shipping.N> }
instance Pushed for shipping.Push
`,
		},
		{
			name: "a different unit",
			src: `
import "dep/shipping.tdl" as shipping
class Wide { width: decimal<shipping.cm> }
instance Wide for shipping.Crate
`,
			want: "shop.shipping.Crate.width is decimal<m>, but Wide requires decimal<shop.shipping.cm>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse("main.tdl", strings.NewReader("package shop.audit\n"+tt.src))
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

// A unit the dependency names through its own import reduces the same way
// in both models.
func TestClassFieldsOfAnImportedTypeThroughATransitiveUnit(t *testing.T) {
	deps := MapLoader{
		"dep/q.tdl": `package q

unit N = kg*m/s^2
`,
		"dep/p.tdl": `package p

import "dep/q.tdl" as q

type Push { force: decimal<q.N> }
`,
	}
	file, err := parser.Parse("main.tdl", strings.NewReader(`package root

import "dep/p.tdl" as p
import "dep/q.tdl" as q

class Pushed { force: decimal<q.N> }
instance Pushed for p.Push
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if _, diags := Lower(file, WithLoader(deps)); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}
