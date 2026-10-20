package contextmapper_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/contextmapper"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("source.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	return model
}

func generate(t *testing.T, m *ir.Model) (string, *plugin.Response) {
	t.Helper()
	resp, err := contextmapper.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: contextmapper.Name,
		Model:  m,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %v, diagnostics = %+v", resp.GetFiles(), resp.GetDiagnostics())
	}
	validate(t, resp.GetFiles()[0])
	return string(resp.GetFiles()[0].GetContent()), resp
}

// validate runs cml-check, the Context Mapper parser and validator, when it
// is on PATH. No CLI ships with Context Mapper; testdata/cmlcheck builds
// one from the context-mapper-dsl jar.
func validate(t *testing.T, f *plugin.File) {
	t.Helper()
	if testing.Short() {
		return
	}
	cli, err := exec.LookPath("cml-check")
	if err != nil {
		return
	}
	path := filepath.Join(t.TempDir(), f.GetPath())
	if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cli, path).CombinedOutput()
	var issues []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.Contains(line, ": ERROR: ") || strings.Contains(line, ": WARNING: ") {
			issues = append(issues, line)
		}
	}
	if err != nil || len(issues) > 0 {
		t.Errorf("Context Mapper rejects the output: %v\n%s\n%s", err, strings.Join(issues, "\n"), f.GetContent())
	}
}

func contains(t *testing.T, src string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(src, want) {
			t.Errorf("output missing %q:\n%s", want, src)
		}
	}
}

func warned(t *testing.T, resp *plugin.Response, wants ...string) {
	t.Helper()
	var msgs []string
	for _, d := range resp.GetDiagnostics() {
		msgs = append(msgs, d.GetMessage())
	}
	all := strings.Join(msgs, "\n")
	for _, want := range wants {
		if !strings.Contains(all, want) {
			t.Errorf("no warning containing %q in:\n%s", want, all)
		}
	}
}

const shop = `/// Taking and paying for orders.
package shop.billing

type OrderId: uuid

/// Something a customer buys.
type Order: Entity {
  id: OrderId
  customer: Customer
  lines: [OrderLine] owned
  total: Money
  status: Status
  type: string
}

type OrderLine: Entity {
  sku: string where { length(1..32) }
  quantity: int where { min(1) max(99) }
  price: Money
}

type Customer: Entity {
  id: uuid
  email: string? where { matches(/^[^@]+@[^@]+$/) }
  credit: Money
}

type Money {
  amount: decimal
  currency: string
}

enum Status {
  Pending
  Paid
}

type OrderPlaced {
  order: Order
  at: instant
}

type PlaceOrder {
  customer: Customer
  lines: [OrderLine]
}

target cml for shop.billing {
  Order => key(id)
  OrderPlaced => event
  PlaceOrder => command
}
`

// Each entity no other entity owns roots an aggregate; an owned entity, and
// a value one aggregate reaches, join it.
func TestAggregates(t *testing.T) {
	src, resp := generate(t, lower(t, shop))
	if len(resp.GetDiagnostics()) > 0 {
		t.Errorf("diagnostics: %+v", resp.GetDiagnostics())
	}
	contains(t, src,
		"// Taking and paying for orders.\nBoundedContext shop_billing {",
		"  Aggregate Order {\n    \"Something a customer buys.\"\n    Entity Order {\n      aggregateRoot\n      - @OrderId id key\n",
		"      - @Customer customer\n      - List<@OrderLine> lines\n",
		"      String ^type\n",
		"    Entity OrderLine {\n      String sku",
		"      String sku notEmpty length = \"32\"\n      Long quantity min = \"1\" max = \"99\"\n",
		"    BasicType OrderId {\n      UUID value\n    }\n",
		"    DomainEvent OrderPlaced {\n",
		"  Aggregate Customer {\n    Entity Customer {\n",
		"      String ^email nullable pattern = \"^[^@]+@[^@]+$\"\n",
		"  Aggregate Shared {\n    ValueObject Money {\n",
		"    CommandEvent PlaceOrder {\n",
		"    enum Status {\n      Pending,\n      Paid\n    }\n",
	)
}

func TestSmoke(t *testing.T) {
	m := irtest.Conformance(t, "../../testdata/gen/smoke")
	src, resp := generate(t, m)
	contains(t, src,
		"  Aggregate Order {\n",
		"    Entity Order {\n      aggregateRoot\n      UUID id\n",
		"      Timestamp createdAt\n      Timestamp updatedAt\n    }\n",
		"    abstract ValueObject Payment {\n    }\n",
		"    ValueObject PaymentCard extends @Payment {\n      String last4\n    }\n",
		"    BasicType WorkEmail {\n      - @Email value\n    }\n",
		"      Set<String> tags\n",
		"      String note nullable\n",
		"    \"Deprecated: use Order\"\n    ValueObject LegacyOrder {\n",
	)
	warned(t, resp,
		"byCurrency is a map",
		"byStatus is a map",
		"values nests collections",
	)
}

func TestGenericsCascade(t *testing.T) {
	_, resp := generate(t, lower(t, `package p

type Page<T> { items: [T] }

type Holder { page: Page<int> }

type Kept: Entity { name: string }
`))
	warned(t, resp, "Page is parameterized", "Holder names Page")
}
