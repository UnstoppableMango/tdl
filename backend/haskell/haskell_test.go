package haskell_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/haskell"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func generate(t *testing.T, m *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := haskell.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: haskell.Name,
		Model:  m,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// check returns the response's one module, type checked by ghc when it is
// on PATH.
func check(t *testing.T, resp *plugin.Response) string {
	t.Helper()
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, diagnostics = %+v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
	f := resp.GetFiles()[0]
	compile(t, f)
	return string(f.GetContent())
}

// compile runs `ghc -fno-code -Wall -Werror` over a module. Without ghc,
// or under -short, these tests check substrings only; checks.gen-haskell
// runs it regardless.
func compile(t *testing.T, f *plugin.File) {
	t.Helper()
	if testing.Short() {
		return
	}
	ghc, err := exec.LookPath("ghc")
	if err != nil {
		return
	}
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(f.GetPath()))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(ghc, "-fno-code", "-Wall", "-Werror", "-haddock", "-Winvalid-haddock", path)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("ghc rejects %s: %v\n%s\n%s", f.GetPath(), err, out, f.GetContent())
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

func absent(t *testing.T, src string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(src, u) {
			t.Errorf("output has %q:\n%s", u, src)
		}
	}
}

// warned reports whether a warning's message contains each substring.
func warned(resp *plugin.Response, subs ...string) bool {
	for _, d := range resp.GetDiagnostics() {
		ok := d.GetSeverity() == plugin.Severity_SEVERITY_WARNING
		for _, s := range subs {
			ok = ok && strings.Contains(d.GetMessage(), s)
		}
		if ok {
			return true
		}
	}
	return false
}

func quiet(t *testing.T, resp *plugin.Response) {
	t.Helper()
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
}

func value(name string, fields ...*ir.Field) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: name},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Kind: ir.StructKind_STRUCT_KIND_VALUE, Fields: fields}},
	}
}

func enum(name string, variants ...*ir.Variant) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: name},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{Variants: variants}},
	}
}

func variant(name string, fields ...*ir.Field) *ir.Variant {
	return &ir.Variant{Meta: &ir.Meta{Name: name}, Fields: fields}
}

func newtype(name string, base *ir.ID) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base}}}
}

func alias(name string, target *ir.ID) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Alias{Alias: &ir.Alias{Target: target}}}
}

func directive(name string, args ...string) *ir.Directive {
	d := &ir.Directive{Name: name, Target: haskell.Name}
	for _, a := range args {
		d.Args = append(d.Args, irtest.Text(a))
	}
	return d
}

func block(dirs ...*ir.Directive) []*ir.TargetBlock {
	return []*ir.TargetBlock{{Meta: &ir.Meta{Name: haskell.Name}, Directives: dirs}}
}

func TestDescribe(t *testing.T) {
	d := haskell.Backend{}.Describe()
	if d.Name != "haskell" || !d.Reuse || d.Reverse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"module", "name", "prefix", "lazy"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared", want)
		}
	}
}

func TestRecords(t *testing.T) {
	b := irtest.New("shop.billing")
	b.Own(value("LineItem", irtest.Field("sku", b.Named("string")), irtest.Field("quantity", b.Named("int"))))
	b.Own(value("Order",
		irtest.Field("id", b.Named("uuid")),
		irtest.Field("items", b.Named("List", b.Named("LineItem"))),
		irtest.Field("tags", b.Named("Set", b.Named("string"))),
		irtest.Field("totals", b.Named("Map", b.Named("string"), b.Named("decimal"))),
		irtest.Field("note", b.Named("Option", b.Named("string"))),
		irtest.Field("shipOn", b.Named("Nullable", b.Named("date"))),
		irtest.Field("either", b.Named("Nullable", b.Named("Option", b.Named("string")))),
		irtest.Field("gaps", b.Named("List", b.Named("Option", b.Named("int")))),
		irtest.Field("nested", b.Named("Map", b.Named("string"), b.Named("Set", b.Named("int32")))),
		irtest.Field("placed", b.Named("instant")),
		irtest.Field("took", b.Named("duration")),
		irtest.Field("receipt", b.Named("bytes")),
		irtest.Field("paid", b.Named("bool")),
		irtest.Field("Weight", b.Named("float64")),
		irtest.Field("ratio", b.Named("float32")),
		irtest.Field("small", b.Named("uint32")),
		irtest.Field("large", b.Named("uint64")),
		irtest.Field("wide", b.Named("int64")),
	))
	b.Own(value("Marker"))

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	if path := resp.GetFiles()[0].GetPath(); path != "Shop/Billing.hs" {
		t.Errorf("path = %q", path)
	}
	contains(t, src,
		"-- Code generated by tdl. DO NOT EDIT.\n{-# LANGUAGE GHC2021 #-}\n",
		"{-# LANGUAGE StrictData #-}\n\nmodule Shop.Billing where\n\n",
		"import Data.ByteString (ByteString)\n",
		"import Data.Int (Int32, Int64)\n",
		"import Data.Map.Strict (Map)\n",
		"import Data.Text (Text)\n",
		"import Data.Time.Calendar (Day)\n",
		"import Data.Time.Clock (NominalDiffTime, UTCTime)\n",
		"import Data.Word (Word32, Word64)\n",
		"import GHC.Generics (Generic)\n",
		"import Prelude (Bool, Double, Eq, Float, Maybe, Ord, Rational, Show)\n",
		"data LineItem = LineItem\n  { sku :: Text\n  , quantity :: Int64\n  }\n  deriving stock (Eq, Ord, Show, Generic)\n",
		"  { id :: Text\n",
		"  , items :: [LineItem]\n",
		"  , tags :: Set Text\n",
		"  , totals :: Map Text Rational\n",
		"  , note :: Maybe Text\n",
		"  , shipOn :: Maybe Day\n",
		"  , either :: Maybe (Maybe Text)\n",
		"  , gaps :: [Maybe Int64]\n",
		"  , nested :: Map Text (Set Int32)\n",
		"  , placed :: UTCTime\n",
		"  , weight :: Double\n",
		"data Marker = Marker\n  deriving stock (Eq, Ord, Show, Generic)\n",
	)
}

func TestEnums(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("draft"), variant("Placed")))
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	b.Own(enum("Shape",
		variant("Circle", irtest.Field("radius", b.Named("float64"))),
		variant("Ring", irtest.Field("radius", b.Named("float64")), irtest.Field("inner", b.Named("float64"))),
	))
	b.Own(enum("Never"))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, check(t, resp),
		"data Status = Draft | Placed\n  deriving stock (Eq, Ord, Show, Read, Enum, Bounded, Generic)\n",
		"data Payment\n  = Card {last4 :: Text}\n  | Cash\n  deriving stock (Eq, Ord, Show, Generic)\n",
		"  | Ring {radius :: Double, inner :: Double}\n",
		"data Never\n  deriving stock (Eq, Ord, Show, Generic)\n",
	)
}

// A field two variants share must have one type, since the enum has one
// field of that label.
func TestASharedVariantFieldNeedsOneType(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Shape",
		variant("Circle", irtest.Field("size", b.Named("float64"))),
		variant("Square", irtest.Field("size", b.Named("int"))),
	))
	b.Own(value("Drawing", irtest.Field("shape", b.Named("Shape"))))
	b.Own(value("Kept", irtest.Field("id", b.Named("string"))))

	resp := generate(t, b.Model)
	if !warned(resp, "Shape.Square.size", "Shape.Circle.size", "different types") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	if !warned(resp, "Drawing names Shape") {
		t.Errorf("Drawing was not skipped: %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	absent(t, src, "Shape", "Drawing")
	contains(t, src, "data Kept")
}

func TestNewtypesAndAliases(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("Email", b.Named("string")))
	b.Own(newtype("WorkEmail", b.Named("Email")))
	b.Own(newtype("Note", b.Named("Option", b.Named("string"))))
	b.Own(alias("Tags", b.Named("Set", b.Named("string"))))
	b.Own(value("Contact", irtest.Field("email", b.Named("WorkEmail")), irtest.Field("tags", b.Named("Tags"))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, check(t, resp),
		"newtype Email = Email Text\n  deriving stock (Eq, Ord, Show, Generic)\n",
		"newtype WorkEmail = WorkEmail Email\n",
		"newtype Note = Note (Maybe Text)\n",
		"type Tags = Set Text\n",
		"  , tags :: Tags\n",
	)
}

func TestModuleName(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, directive, path, err string
	}{
		{name: "package", pkg: "shop.v1_api", path: "Shop/V1_api.hs"},
		{name: "directive", pkg: "shop", directive: "Acme.Billing", path: "Acme/Billing.hs"},
		{name: "directive without a package", directive: "Acme", path: "Acme.hs"},
		{name: "none", err: "needs a name"},
		{name: "refused", pkg: "shop", directive: "acme.billing", err: "acme.billing is not a Haskell module name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := irtest.New(tc.pkg)
			b.Own(value("Order", irtest.Field("id", b.Named("string"))))
			if tc.directive != "" {
				b.Model.Targets = block(directive("module", tc.directive))
			}
			resp := generate(t, b.Model)
			if tc.err != "" {
				diags := resp.GetDiagnostics()
				if len(resp.GetFiles()) != 0 || len(diags) != 1 ||
					diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), tc.err) {
					t.Errorf("files = %v, diagnostics = %+v", resp.GetFiles(), diags)
				}
				return
			}
			check(t, resp)
			if path := resp.GetFiles()[0].GetPath(); path != tc.path {
				t.Errorf("path = %q, want %q", path, tc.path)
			}
		})
	}
}

func TestNameDirective(t *testing.T) {
	b := irtest.New("shop")
	f := irtest.Field("id", b.Named("string"))
	f.Directives = []*ir.Directive{directive("name", "orderId")}
	order := value("Order", f)
	order.Directives = []*ir.Directive{directive("name", "Purchase")}
	b.Own(order)
	v := variant("Draft")
	v.Directives = []*ir.Directive{directive("name", "Pending")}
	b.Own(enum("Status", v, variant("Placed")))
	b.Own(value("Ledger", irtest.Field("order", b.Named("Order"))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, check(t, resp),
		"data Purchase = Purchase\n  { orderId :: Text\n",
		"data Status = Pending | Placed\n",
		"  { order :: Purchase\n",
	)
}

func TestPrefix(t *testing.T) {
	b := irtest.New("shop")
	status := enum("Status", variant("Draft"), variant("Placed"))
	status.Directives = []*ir.Directive{{Name: "prefix", Target: haskell.Name}}
	b.Own(status)
	b.Own(enum("Payment", variant("Card"), variant("Cash")))

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src, "data Status = StatusDraft | StatusPlaced\n", "data Payment = Card | Cash\n")

	b.Model.Targets = block(&ir.Directive{Name: "prefix", Target: haskell.Name})
	contains(t, check(t, generate(t, b.Model)), "data Payment = PaymentCard | PaymentCash\n")
}

// A constructor shares one namespace across the module, so a later
// declaration taking a constructor is skipped, along with whatever names
// it.
func TestConstructorCollisions(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Card", irtest.Field("last4", b.Named("string"))))
	b.Own(enum("Payment", variant("Card"), variant("Cash")))
	b.Own(value("Checkout", irtest.Field("payment", b.Named("Payment"))))
	b.Own(enum("Doubled", variant("Same"), variant("Same")))

	resp := generate(t, b.Model)
	if !warned(resp, "Payment would declare the constructor Card", "Card already does") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	if !warned(resp, "Checkout names Payment") {
		t.Errorf("Checkout was not skipped: %+v", resp.GetDiagnostics())
	}
	if !warned(resp, "Doubled would declare the constructor Same") {
		t.Errorf("Doubled was not skipped: %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src, "data Card = Card")
	absent(t, src, "Payment", "Checkout", "Doubled")

	// prefix resolves it.
	b.Model.Targets = block(&ir.Directive{Name: "prefix", Target: haskell.Name})
	resp = generate(t, b.Model)
	src = check(t, resp)
	contains(t, src, "data Payment = PaymentCard | PaymentCash\n")
	absent(t, src, "Doubled")
}

func TestTypeNameCollisions(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Order"))
	second := value("Purchase")
	second.Directives = []*ir.Directive{directive("name", "Order")}
	b.Own(second)
	b.Own(value("lower", irtest.Field("id", b.Named("string"))))
	bad := value("Bad")
	bad.Directives = []*ir.Directive{directive("name", "not-a-name")}
	b.Own(bad)

	resp := generate(t, b.Model)
	if !warned(resp, "Purchase would declare the type Order", "Order already does") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	if !warned(resp, "Bad would be named not-a-name") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, check(t, resp), "data Lower = Lower\n")
}

func TestKeywordsAreEscaped(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Token",
		irtest.Field("type", b.Named("string")),
		irtest.Field("Data", b.Named("string")),
		irtest.Field("module", b.Named("string")),
	))
	b.Own(value("Clash", irtest.Field("type", b.Named("string")), irtest.Field("type_", b.Named("string"))))

	resp := generate(t, b.Model)
	if !warned(resp, "Clash has two fields labelled type_") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src, "  { type_ :: Text\n  , data_ :: Text\n  , module_ :: Text\n")
	absent(t, src, "Clash")
}

// A declaration named like an imported type makes that import qualified.
func TestADeclaredNameQualifiesItsImport(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Text", irtest.Field("body", b.Named("string"))))
	b.Own(value("Show", irtest.Field("title", b.Named("Text")), irtest.Field("live", b.Named("bool"))))
	b.Own(value("Set", irtest.Field("members", b.Named("Set", b.Named("int")))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"import Data.Set qualified as Set\n",
		"import Data.Text qualified as Text\n",
		"import Prelude qualified as Prelude\n",
		"data Text = Text\n  { body :: Text.Text\n",
		"  { title :: Text\n  , live :: Prelude.Bool\n",
		"  { members :: Set.Set Int64\n",
		"  deriving stock (Prelude.Eq, Prelude.Ord, Prelude.Show, Generic)\n",
	)
	absent(t, src, "import Data.Text (")
}

func TestLazy(t *testing.T) {
	b := irtest.New("shop")
	f := irtest.Field("children", b.Named("Set", b.Named("int")))
	f.Directives = []*ir.Directive{{Name: "lazy", Target: haskell.Name}}
	g := irtest.Field("parent", b.Named("string"))
	g.Directives = []*ir.Directive{{Name: "lazy", Target: haskell.Name}}
	b.Own(value("Node", f, g))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, check(t, resp), "  { children :: ~(Set Int64)\n  , parent :: ~Text\n")
}

func TestDocsAndDeprecation(t *testing.T) {
	b := irtest.New("shop")
	b.Model.Doc = []string{"The shop."}

	fax := irtest.Field("fax", b.Named("string"))
	fax.Meta.Deprecated = &ir.Deprecation{Reason: "nobody has one"}
	email := irtest.Field("email", b.Named("string"))
	email.Meta.Doc = []string{"Where to write.", "", "Checked on signup."}
	contact := value("Contact", email, fax)
	contact.Meta.Doc = []string{"How to reach someone."}
	contact.Meta.Deprecated = &ir.Deprecation{}
	b.Own(contact)

	// fax is a label two records share, so its pragma would be ambiguous.
	shared := irtest.Field("pager", b.Named("string"))
	shared.Meta.Deprecated = &ir.Deprecation{Reason: "gone"}
	b.Own(value("Office", shared))
	b.Own(value("Desk", irtest.Field("pager", b.Named("string"))))

	draft := variant("Draft")
	draft.Meta.Doc = []string{"Not yet placed.", "Editable."}
	old := variant("Provisional")
	old.Meta.Deprecated = &ir.Deprecation{Reason: `say "Placed"`}
	b.Own(enum("Status", draft, variant("Placed"), old))

	card := irtest.Field("last4", b.Named("string"))
	card.Meta.Doc = []string{"The last four digits."}
	b.Own(enum("Payment", variant("Card", card, irtest.Field("brand", b.Named("string"))), variant("Cash")))

	resp := generate(t, b.Model)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"-- | The shop.\nmodule Shop where\n",
		"-- | How to reach someone.\ndata Contact = Contact\n",
		"  { email :: Text\n    -- ^ Where to write.\n    --\n    -- Checked on signup.\n  , fax :: Text\n    -- ^ Deprecated: nobody has one\n  }\n",
		"{-# DEPRECATED Contact \"deprecated\" #-}\n",
		"{-# DEPRECATED fax \"nobody has one\" #-}\n",
		"  { pager :: Text\n    -- ^ Deprecated: gone\n",
		"data Status\n  = -- | Not yet placed.\n    -- Editable.\n    Draft\n  | Placed\n  | Provisional\n",
		`{-# DEPRECATED Provisional "say \"Placed\"" #-}`,
		"  = Card\n      { last4 :: Text\n        -- ^ The last four digits.\n      , brand :: Text\n      }\n  | Cash\n",
	)
	absent(t, src, "DEPRECATED pager")
}

// What later phases generate warns, and is skipped with whatever names it.
func TestNotYetGenerated(t *testing.T) {
	b := irtest.New("shop")
	b.Decl(&ir.Decl{Meta: &ir.Meta{Name: "kg"}, Node: &ir.Decl_Unit{Unit: &ir.UnitDef{}}})
	b.Class("Auditable")
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Page"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Params: irtest.Params("T"),
			Fields: []*ir.Field{irtest.Field("items", b.Named("List", b.Param("T", 0)))},
		}},
	})
	b.Own(value("Parcel", irtest.Field("weight", b.Unit("kg", 3))))
	b.Own(value("Invoice", irtest.Field("money", b.Extern("money.Money"))))
	b.Own(alias("Cash", b.Extern("money.Money")))
	b.Own(value("Shipment", irtest.Field("parcel", b.Named("Parcel"))))
	b.Own(value("Kept", irtest.Field("id", b.Named("string"))))

	resp := generate(t, b.Model)
	for _, want := range []struct{ code, msg string }{
		{emit.LossClass, "Auditable is a class"},
		{emit.LossGeneric, "Page is parameterized"},
		{emit.LossUnit, "unit-typed field"},
		{"", "money.Money is declared in another package"},
		{"", "Shipment names Parcel"},
	} {
		found := false
		for _, d := range resp.GetDiagnostics() {
			found = found || (d.GetCode() == want.code && strings.Contains(d.GetMessage(), want.msg))
		}
		if !found {
			t.Errorf("no %q warning %q: %+v", want.code, want.msg, resp.GetDiagnostics())
		}
	}
	src := check(t, resp)
	absent(t, src, "Page", "Parcel", "Invoice", "Cash", "Shipment", "Auditable")
	contains(t, src, "data Kept")
}

func TestConstraintsWarn(t *testing.T) {
	b := irtest.New("shop")
	f := irtest.Field("quantity", b.Named("int"))
	f.Constraints = []*ir.Constraint{{Name: "min", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "1"}}}}
	b.Own(value("LineItem", f))

	resp := generate(t, b.Model)
	if !warned(resp, "LineItem.quantity carries 1 constraint(s)") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, check(t, resp), "quantity :: Int64")
}

func TestAnEmptyModelWritesNothing(t *testing.T) {
	resp := generate(t, irtest.New("shop").Model)
	if len(resp.GetFiles()) != 0 || len(resp.GetDiagnostics()) != 0 {
		t.Errorf("response = %+v", resp)
	}
}

// The smoke model generates one module, which checks.gen-haskell compiles.
func TestSmoke(t *testing.T) {
	m := irtest.Conformance(t, "../../testdata/gen/smoke")
	resp := generate(t, m)
	for _, d := range resp.GetDiagnostics() {
		if d.GetCode() != emit.LossConstraint {
			t.Errorf("diagnostic: %+v", d)
		}
	}
	src := check(t, resp)
	if path := resp.GetFiles()[0].GetPath(); path != "Smoke.hs" {
		t.Errorf("path = %q", path)
	}
	contains(t, src, "data Order = Order\n", "type Tags = Set Text\n", "data Status = Draft | Placed | Provisional\n")
}
