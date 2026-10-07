package sema

import (
	"strings"
	"testing"
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
