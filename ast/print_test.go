package ast_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func mustParse(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return file
}

func TestFprintCanonical(t *testing.T) {
	src := `package example.aliases

import "common.tdl" as common

/// A handler table.
primitive string

primitive Map: type -> type -> type
primitive Higher: (type -> type) -> type
alias Applied<f: type -> type, T> = f<T>
alias Handler = {string -> [Event]}
alias Both = LineItem? | null
alias Qualified = Map<string, common.Address>

/// A distinct type over another.
type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
}

type Slug: string where { length(1..64) }

entity Order {
  key id: OrderId
  customer: Customer
  items: [LineItem] owned
  status: Status = Draft
  quantity: int where { min(1) }
  deprecated legacy: string
}

value Money {
  amount: decimal
  currency: Currency
}

mixin Timestamps {
  include Auditable
  createdAt: instant
}

enum Status { Draft Placed Shipped }

enum Payment {
  Card { last4: string brand: CardBrand }
  Credit
}

deprecated("use Contact")
entity LegacyContact {
  key id: string
}

target go for example.aliases {
  out("./gen/go")
  package("github.com/acme/x")
  Order {
    name("PurchaseOrder")
    customer => tag("json:buyer")
  }
  Order.items => slice
  Money => foreign("github.com/shopspring/decimal", "Decimal")
}
`

	got := ast.Fprint(mustParse(t, src))
	if got != src {
		t.Errorf("Fprint mismatch\n--- got ---\n%s\n--- want ---\n%s", got, src)
	}
}

// Formatting canonical output must change nothing, and formatting anything
// else must reach canonical output in one pass.
func TestFprintIdempotent(t *testing.T) {
	messy := `package   p
primitive string primitive int
alias   A=[  T ]
alias B = { K->V }
entity  E{key id:string
  tags:{string}=[]
  n:int where{min(0) max(10)}}
enum Big { AlphaVariant BetaVariant GammaVariant DeltaVariant EpsilonVariant ZetaVariant }
`

	once := ast.Fprint(mustParse(t, messy))
	twice := ast.Fprint(mustParse(t, once))
	if once != twice {
		t.Errorf("not idempotent\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}
