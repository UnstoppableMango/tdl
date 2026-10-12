package haskell_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

func entity(name string, fields ...*ir.Field) *ir.Decl {
	d := value(name, fields...)
	d.GetStructure().Kind = ir.StructKind_STRUCT_KIND_ENTITY
	return d
}

// keyOn is a key directive naming fields.
func keyOn(fields ...string) *ir.Directive {
	d := &ir.Directive{Name: "key", Target: "haskell"}
	for _, f := range fields {
		d.Args = append(d.Args, irtest.Name(f))
	}
	return d
}

func TestKeys(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("UserId", b.Named("string")))
	user := entity("User", irtest.Field("id", b.Named("UserId")), irtest.Field("name", b.Named("string")))
	user.Directives = []*ir.Directive{keyOn("id")}
	b.Own(user)
	b.Own(newtype("Sku", b.Named("string")))
	typ := irtest.Field("type", b.Named("string"))
	typ.Meta.Deprecated = &ir.Deprecation{Reason: "gone"}
	line := entity("LineItem",
		irtest.Field("order", b.Named("string")),
		irtest.Field("sku", b.Named("Sku")),
		typ,
		irtest.Field("quantity", b.Named("int")),
	)
	line.Directives = []*ir.Directive{keyOn("order", "sku", "type")}
	b.Own(line)

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"-- | What identifies 'User' values.\nuserKey :: User -> UserId\nuserKey u = u.id\n",
		"data LineItemKey = LineItemKey {order :: Text, sku :: Sku, type_ :: Text}\n  deriving stock (Eq, Ord, Show, Generic)\n",
		"lineItemKey :: LineItem -> LineItemKey\nlineItemKey l = LineItemKey {order = l.order, sku = l.sku, type_ = l.type_}\n",
	)
	// type_ is a label the key record shares, so its pragma would be
	// ambiguous.
	absent(t, src, "DEPRECATED type_")
}

func TestKeysThatCannotBeGenerated(t *testing.T) {
	for _, tc := range []struct {
		name string
		decl func(b *irtest.Builder) *ir.Decl
		want string
	}{
		{
			name: "value",
			decl: func(b *irtest.Builder) *ir.Decl {
				d := value("Order", irtest.Field("id", b.Named("string")))
				d.Directives = []*ir.Directive{keyOn("id")}
				return d
			},
			want: "only an entity is identified by one",
		},
		{
			name: "missing field",
			decl: func(b *irtest.Builder) *ir.Decl {
				d := entity("Order", irtest.Field("id", b.Named("string")))
				d.Directives = []*ir.Directive{keyOn("number")}
				return d
			},
			want: "Order has no field number to key on",
		},
		{
			name: "repeated field",
			decl: func(b *irtest.Builder) *ir.Decl {
				d := entity("Order", irtest.Field("id", b.Named("string")))
				d.Directives = []*ir.Directive{keyOn("id", "id")}
				return d
			},
			want: "names id twice",
		},
		{
			name: "record collision",
			decl: func(b *irtest.Builder) *ir.Decl {
				d := entity("Order", irtest.Field("id", b.Named("string")), irtest.Field("rev", b.Named("int")))
				d.Directives = []*ir.Directive{keyOn("id", "rev")}
				b.Own(d)
				return value("OrderKey", irtest.Field("raw", b.Named("string")))
			},
			want: "the key record OrderKey would collide with OrderKey",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := irtest.New("shop")
			b.Own(tc.decl(b))
			resp := generate(t, b.Model)
			found := false
			for _, d := range resp.GetDiagnostics() {
				found = found || (d.GetCode() == emit.LossKey && strings.Contains(d.GetMessage(), tc.want))
			}
			if !found {
				t.Errorf("no key warning %q: %+v", tc.want, resp.GetDiagnostics())
			}
			src := check(t, resp)
			contains(t, src, "data Order")
			absent(t, src, "orderKey")
		})
	}
}

func TestForeign(t *testing.T) {
	b := irtest.New("shop")
	// A primitive mapped by a target path, as `uuid => foreign(...)` is.
	_, id, _ := b.Model.FindDecl("uuid")
	b.Model.Decls[id.GetIndex()].Directives = []*ir.Directive{directive("foreign", "Data.Fixed", "Centi")}
	money := value("Money", irtest.Field("units", b.Named("int")))
	money.Directives = []*ir.Directive{directive("foreign", "Numeric.Natural", "Natural")}
	b.Own(money)
	// A foreign type named like a declared one is imported qualified.
	text := value("Label")
	text.Directives = []*ir.Directive{directive("foreign", "Data.Text", "Text")}
	b.Own(text)
	b.Own(value("Text", irtest.Field("body", b.Named("string"))))
	b.Own(value("Order",
		irtest.Field("id", b.Named("uuid")),
		irtest.Field("total", b.Named("Money")),
		irtest.Field("totals", b.Named("List", b.Named("Money"))),
		irtest.Field("label", b.Named("Label")),
		irtest.Field("text", b.Named("Text")),
	))

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"import Data.Fixed (Centi)\n",
		"import Data.Text qualified as Text\n",
		"import Numeric.Natural (Natural)\n",
		"  { id :: Centi\n  , total :: Natural\n  , totals :: [Natural]\n  , label :: Text.Text\n  , text :: Text\n",
		"data Text = Text\n  { body :: Text.Text\n",
	)
	absent(t, src, "data Money", "data Label")
}

func TestForeignThatCannotBeUsed(t *testing.T) {
	b := irtest.New("shop")
	bad := value("Money", irtest.Field("units", b.Named("int")))
	bad.Directives = []*ir.Directive{directive("foreign", "numeric", "Natural")}
	b.Own(bad)
	q := irtest.Field("units", b.Named("int"))
	q.Constraints = []*ir.Constraint{{Name: "min", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "0"}}}}
	keyed := entity("Account", irtest.Field("id", b.Named("string")), q)
	keyed.Directives = []*ir.Directive{directive("foreign", "Numeric.Natural", "Natural"), keyOn("id")}
	b.Own(keyed)

	resp := generate(t, b.Model)
	for _, want := range []string{
		`Money is mapped into "numeric", which is not a Haskell module name`,
		"Account is a foreign type, so its key is not generated",
		"min(0) is not checked: Account is a foreign type",
	} {
		if !warned(resp, want) {
			t.Errorf("no warning %q: %+v", want, resp.GetDiagnostics())
		}
	}
	src := check(t, resp)
	contains(t, src, "data Money = Money\n")
	absent(t, src, "Account")
}

func TestForeignExtern(t *testing.T) {
	b := irtest.New("shop")
	money := b.ExternIn("acme.money", "Money", directive("foreign", "Numeric.Natural", "Natural"))
	b.Own(value("Order", irtest.Field("total", money)))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, check(t, resp), "import Numeric.Natural (Natural)\n", "  { total :: Natural\n")
}

func TestDerive(t *testing.T) {
	b := irtest.New("shop")
	b.Model.Targets = block(directive("derive", "Control.DeepSeq", "NFData"))
	order := value("Order", irtest.Field("id", b.Named("string")))
	order.Directives = []*ir.Directive{
		directive("derive", "Control.DeepSeq", "NFData"),
		directive("derive", "Data.Hashable", "hashable"),
	}
	b.Own(order)
	b.Own(newtype("Email", b.Named("string")))
	b.Own(enum("Status", variant("Draft"), variant("Placed")))
	tags := alias("Tags", b.Named("Set", b.Named("string")))
	tags.Directives = []*ir.Directive{directive("derive", "Control.DeepSeq", "NFData")}
	b.Own(tags)

	resp := generate(t, b.Model)
	if !warned(resp, `derive names "hashable", which is not a Haskell class name`) ||
		!warned(resp, "Tags is an alias") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src,
		"{-# LANGUAGE StrictData #-}\n{-# LANGUAGE DeriveAnyClass #-}\n",
		"import Control.DeepSeq (NFData)\n",
		"  deriving stock (Eq, Ord, Show, Generic)\n  deriving anyclass (NFData)\n\nnewtype Email",
		"newtype Email = Email Text\n  deriving stock (Eq, Ord, Show, Generic)\n  deriving anyclass (NFData)\n",
		"  deriving stock (Eq, Ord, Show, Read, Enum, Bounded, Generic)\n  deriving anyclass (NFData)\n",
	)
	if strings.Count(src, "NFData") != 4 {
		t.Errorf("NFData is derived more or less than once per declaration:\n%s", src)
	}
}

// deps is an in-memory [sema.Loader] keyed by import path.
type deps map[string]string

func (d deps) Load(_, path string) (string, string, error) {
	src, ok := d[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}

func lowerSource(t *testing.T, name, src string, d deps) *ir.Model {
	t.Helper()
	file, err := parser.Parse(name, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(d))
	if len(diags) > 0 {
		t.Fatalf("lowering %s: %v", name, diags.Error())
	}
	return model
}

// compileTogether runs ghc over every module the responses hold, as one
// source tree.
func compileTogether(t *testing.T, resps ...*plugin.Response) {
	t.Helper()
	if testing.Short() {
		return
	}
	ghc, err := exec.LookPath("ghc")
	if err != nil {
		return
	}
	dir := t.TempDir()
	var paths, all []string
	for _, resp := range resps {
		for _, f := range resp.GetFiles() {
			path := filepath.Join(dir, filepath.FromSlash(f.GetPath()))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
			all = append(all, "==> "+f.GetPath()+" <==\n"+string(f.GetContent()))
		}
	}
	cmd := exec.Command(ghc, append([]string{"-fno-code", "-Wall", "-Werror", "-i" + dir}, paths...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("ghc rejects the modules: %v\n%s\n%s", err, out, strings.Join(all, "\n"))
	}
}

const moneySource = `package acme.money

type Money { units: int }

type Currency { code: string }

target haskell for acme.money {
  module("Acme.Currency")
  Currency => name("ISO4217")
}
`

// A two-package model generated into one out compiles as one set of
// modules: a reference into the dependency imports the module its haskell
// target block names.
func TestDependencyWithHaskellBlockIsImported(t *testing.T) {
	d := deps{"money.tdl": moneySource}
	root := lowerSource(t, "shop.tdl", `package shop

import "money.tdl" as money

type Order {
  total: money.Money
  currency: money.Currency
}

target haskell for shop { }
`, d)
	resp := generate(t, root)
	quiet(t, resp)
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %v", resp.GetFiles())
	}
	contains(t, string(resp.GetFiles()[0].GetContent()),
		"import Acme.Currency (ISO4217, Money)\n",
		"  { total :: Money\n  , currency :: ISO4217\n",
	)
	dep := generate(t, lowerSource(t, "money.tdl", moneySource, nil))
	compileTogether(t, resp, dep)
}

func TestDependencyWithoutHaskellBlockIsSkipped(t *testing.T) {
	d := deps{"money.tdl": "package acme.money\n\ntype Money { units: int }\n"}
	root := lowerSource(t, "shop.tdl", `package shop

import "money.tdl" as money

type Order { total: money.Money }

type Cart { order: Order }

type Kept { id: string }
`, d)
	resp := generate(t, root)
	if !warned(resp, "acme.money.Money is declared in another package, which has no haskell target block") ||
		!warned(resp, "Cart names Order") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src, "data Kept")
	absent(t, src, "Order", "Cart")
}
