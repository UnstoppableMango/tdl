package csharp_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/csharp"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func generate(t *testing.T, m *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := csharp.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: csharp.Name,
		Model:  m,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// file returns the response's file at path.
func file(t *testing.T, resp *plugin.Response, path string) string {
	t.Helper()
	for _, f := range resp.GetFiles() {
		if f.GetPath() == path {
			return string(f.GetContent())
		}
	}
	var paths []string
	for _, f := range resp.GetFiles() {
		paths = append(paths, f.GetPath())
	}
	t.Fatalf("no file %s among %v; diagnostics = %+v", path, paths, resp.GetDiagnostics())
	return ""
}

// dotnet returns the dotnet CLI, or "" under -short or when it is not on
// PATH. checks.gen-csharp builds the smoke model regardless.
func dotnet(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		return ""
	}
	path, err := exec.LookPath("dotnet")
	if err != nil {
		return ""
	}
	return path
}

// project writes the response's files into a project directory with the
// check project, and program as Program.cs when it is not empty.
func project(t *testing.T, resp *plugin.Response, program string) string {
	t.Helper()
	dir := t.TempDir()
	proj, err := os.ReadFile("testdata/check.csproj")
	if err != nil {
		t.Fatal(err)
	}
	if program != "" {
		proj = []byte(strings.Replace(string(proj), "<PropertyGroup>", "<PropertyGroup>\n    <OutputType>Exe</OutputType>", 1))
		if err := os.WriteFile(filepath.Join(dir, "Program.cs"), []byte(program), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "check.csproj"), proj, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range resp.GetFiles() {
		path := filepath.Join(dir, filepath.FromSlash(f.GetPath()))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, f.GetContent(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func command(bin, dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"DOTNET_CLI_TELEMETRY_OPTOUT=1",
		"DOTNET_NOLOGO=1",
		"DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1",
	)
	return cmd
}

// build runs dotnet build over the response with warnings as errors.
func build(t *testing.T, resp *plugin.Response) {
	t.Helper()
	bin := dotnet(t)
	if bin == "" {
		return
	}
	dir := project(t, resp, "")
	if out, err := command(bin, dir, "build").CombinedOutput(); err != nil {
		t.Errorf("dotnet build rejects the response: %v\n%s\n%s", err, out, dump(resp))
	}
}

// run builds the response with program as its entry point and returns what
// it prints, skipping the test without dotnet.
func run(t *testing.T, resp *plugin.Response, program string) string {
	t.Helper()
	bin := dotnet(t)
	if bin == "" {
		t.Skip("dotnet is not on PATH, or -short")
	}
	dir := project(t, resp, program)
	out, err := command(bin, dir, "run").CombinedOutput()
	if err != nil {
		t.Fatalf("dotnet run: %v\n%s\n%s", err, out, dump(resp))
	}
	return string(out)
}

func dump(resp *plugin.Response) string {
	var b strings.Builder
	for _, f := range resp.GetFiles() {
		b.WriteString("==> " + f.GetPath() + " <==\n")
		b.Write(f.GetContent())
	}
	return b.String()
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
	d := &ir.Directive{Name: name, Target: csharp.Name}
	for _, a := range args {
		d.Args = append(d.Args, irtest.Text(a))
	}
	return d
}

func block(dirs ...*ir.Directive) []*ir.TargetBlock {
	return []*ir.TargetBlock{{Meta: &ir.Meta{Name: csharp.Name}, Directives: dirs}}
}

func named(d *ir.Decl, dirs ...*ir.Directive) *ir.Decl {
	d.Directives = append(d.Directives, dirs...)
	return d
}

func TestDescribe(t *testing.T) {
	d := csharp.Backend{}.Describe()
	if d.Name != "csharp" || !d.Reuse || d.Reverse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"namespace", "file", "name"} {
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
		irtest.Field("gaps", b.Named("List", b.Named("Option", b.Named("int")))),
		irtest.Field("nested", b.Named("Map", b.Named("string"), b.Named("Set", b.Named("int32")))),
		irtest.Field("placed", b.Named("instant")),
		irtest.Field("took", b.Named("duration")),
		irtest.Field("receipt", b.Named("bytes")),
		irtest.Field("paid", b.Named("bool")),
		irtest.Field("weight", b.Named("float64")),
		irtest.Field("ratio", b.Named("float32")),
		irtest.Field("small", b.Named("uint32")),
		irtest.Field("large", b.Named("uint64")),
		irtest.Field("wide", b.Named("int64")),
		irtest.Field("httpURL", b.Named("string")),
	))
	b.Own(value("Marker"))

	resp := generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	src := file(t, resp, "Order.g.cs")
	contains(t, src,
		"// <auto-generated>\n// Code generated by tdl. DO NOT EDIT.\n// </auto-generated>\n#nullable enable\n",
		"\nnamespace Shop.Billing;\n\npublic sealed partial record Order\n{\n",
		"    public required global::System.Guid Id { get; init; }\n",
		"    public required global::System.Collections.Immutable.IImmutableList<LineItem> Items { get; init; }\n",
		"    public required global::System.Collections.Immutable.IImmutableSet<string> Tags { get; init; }\n",
		"    public required global::System.Collections.Immutable.IImmutableDictionary<string, decimal> Totals { get; init; }\n",
		"    public string? Note { get; init; }\n",
		"    public required global::System.DateOnly? ShipOn { get; init; }\n",
		"    public required global::System.Collections.Immutable.IImmutableList<long?> Gaps { get; init; }\n",
		"IImmutableDictionary<string, global::System.Collections.Immutable.IImmutableSet<int>> Nested",
		"    public required global::System.DateTimeOffset Placed { get; init; }\n",
		"    public required global::System.TimeSpan Took { get; init; }\n",
		"    public required byte[] Receipt { get; init; }\n",
		"    public required bool Paid { get; init; }\n",
		"    public required double Weight { get; init; }\n",
		"    public required float Ratio { get; init; }\n",
		"    public required uint Small { get; init; }\n",
		"    public required ulong Large { get; init; }\n",
		"    public required long Wide { get; init; }\n",
		"    public required string HttpURL { get; init; }\n",
		"    public bool Equals(Order? other) =>\n",
		"    public override int GetHashCode()\n",
	)
	// A record holding no collection keeps the compiler's equality.
	absent(t, file(t, resp, "LineItem.g.cs"), "Equals", "TdlEquality")
	contains(t, file(t, resp, "LineItem.g.cs"), "    public required long Quantity { get; init; }\n")
	contains(t, file(t, resp, "Marker.g.cs"), "public sealed partial record Marker;\n")
	contains(t, file(t, resp, "TdlEquality.g.cs"), "internal static class TdlEquality\n")
}

func TestNoEqualityClassWithoutCollections(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Point", irtest.Field("x", b.Named("int"))))
	resp := generate(t, b.Model)
	quiet(t, resp)
	if len(resp.GetFiles()) != 1 {
		t.Errorf("files = %d, want only Point.g.cs", len(resp.GetFiles()))
	}
}

func TestOptionality(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Contact",
		irtest.Field("either", b.Named("Nullable", b.Named("Option", b.Named("string")))),
	))
	resp := generate(t, b.Model)
	build(t, resp)
	contains(t, file(t, resp, "Contact.g.cs"), "    public string? Either { get; init; }\n")
	if !warned(resp, "cannot tell an absent value from a null one") {
		t.Errorf("T? | null does not warn: %+v", resp.GetDiagnostics())
	}
}

func TestEnums(t *testing.T) {
	b := irtest.New("shop")
	provisional := variant("Provisional")
	provisional.Meta.Deprecated = &ir.Deprecation{Reason: `no longer "issued"`}
	draft := variant("Draft")
	draft.Meta.Doc = []string{"Not yet placed."}
	b.Own(enum("Status", draft, variant("placed"), provisional))
	card := variant("Card", irtest.Field("last4", b.Named("string")))
	card.Meta.Doc = []string{"Paid by card."}
	payment := enum("Payment", card, variant("Cash"))
	b.Own(payment)
	split := variant("Split", irtest.Field("parts", b.Named("List", b.Named("Payment"))))
	payment.GetEnumeration().Variants = append(payment.GetEnumeration().Variants, split)

	resp := generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	contains(t, file(t, resp, "Status.g.cs"),
		"public enum Status\n{\n    /// <summary>Not yet placed.</summary>\n    Draft,\n    Placed,\n    [global::System.Obsolete(\"no longer \\\"issued\\\"\")]\n    Provisional,\n}\n")
	contains(t, file(t, resp, "Payment.g.cs"),
		"public abstract partial record Payment\n{\n    private Payment() { }\n",
		"\n    /// <summary>Paid by card.</summary>\n    public sealed partial record Card : Payment\n    {\n        public required string Last4 { get; init; }\n    }\n",
		"\n    public sealed partial record Cash : Payment;\n",
		"    public sealed partial record Split : Payment\n",
		"        public bool Equals(Split? other) =>\n",
	)
}

func TestNewtypesAndAliases(t *testing.T) {
	b := irtest.New("shop")
	email := newtype("Email", b.Named("string"))
	email.Meta.Doc = []string{"An address to write to."}
	b.Own(email)
	b.Own(newtype("WorkEmail", b.Named("Email")))
	b.Own(newtype("Skus", b.Named("List", b.Named("string"))))
	b.Own(alias("Tags", b.Named("Set", b.Named("string"))))
	b.Own(value("Order",
		irtest.Field("contact", b.Named("WorkEmail")),
		irtest.Field("tags", b.Named("Tags")),
		irtest.Field("skus", b.Named("Skus")),
	))

	resp := generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	contains(t, file(t, resp, "Email.g.cs"),
		"/// <summary>An address to write to.</summary>\npublic readonly partial record struct Email(string Value);\n")
	contains(t, file(t, resp, "WorkEmail.g.cs"), "public readonly partial record struct WorkEmail(Email Value);\n")
	contains(t, file(t, resp, "Skus.g.cs"),
		"public readonly partial record struct Skus(global::System.Collections.Immutable.IImmutableList<string> Value)\n{\n",
		"    public bool Equals(Skus other) => _value.Equals(this.Value, other.Value);\n")
	// An alias is expanded where it is used.
	contains(t, file(t, resp, "Order.g.cs"), "public required global::System.Collections.Immutable.IImmutableSet<string> Tags { get; init; }\n")
	for _, f := range resp.GetFiles() {
		if f.GetPath() == "Tags.g.cs" {
			t.Error("an alias declares a type")
		}
	}
}

func TestDeprecationAndDocs(t *testing.T) {
	b := irtest.New("shop")
	fax := irtest.Field("fax", b.Named("string"))
	fax.Meta.Deprecated = &ir.Deprecation{}
	fax.Meta.Doc = []string{"Where to send <faxes> & more.", "Rarely used."}
	legacy := value("LegacyOrder", irtest.Field("id", b.Named("string")), fax)
	legacy.Meta.Deprecated = &ir.Deprecation{Reason: "use Order"}
	b.Own(legacy)
	b.Own(value("Holder", irtest.Field("old", b.Named("LegacyOrder"))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	contains(t, file(t, resp, "LegacyOrder.g.cs"),
		"[global::System.Obsolete(\"use Order\")]\npublic sealed partial record LegacyOrder\n",
		"    public required string Id { get; init; }\n\n"+
			"    /// <summary>\n    /// Where to send &lt;faxes&gt; &amp; more.\n    /// Rarely used.\n    /// </summary>\n"+
			"    [global::System.Obsolete]\n    public required string Fax { get; init; }\n",
	)
}

func TestFileDirective(t *testing.T) {
	b := irtest.New("shop")
	b.Model.Targets = block(directive("file", "Model.g.cs"))
	b.Own(value("LineItem", irtest.Field("sku", b.Named("string"))))
	b.Own(value("Order", irtest.Field("items", b.Named("List", b.Named("LineItem")))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, want 1", len(resp.GetFiles()))
	}
	src := file(t, resp, "Model.g.cs")
	if strings.Count(src, "namespace Shop;") != 1 {
		t.Errorf("one file declares its namespace once:\n%s", src)
	}
	contains(t, src, "public sealed partial record Order\n", "public sealed partial record LineItem\n", "internal static class TdlEquality\n")
}

func TestNamespace(t *testing.T) {
	b := irtest.New("shop.billing")
	b.Model.Targets = block(directive("namespace", "Acme.Billing.V1"))
	b.Own(value("Order", irtest.Field("id", b.Named("string"))))
	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, file(t, resp, "Order.g.cs"), "\nnamespace Acme.Billing.V1;\n")

	for _, tc := range []struct{ pkg, ns, want string }{
		{"", "", "needs a name"},
		{"shop", "Acme.class", "not a C# namespace"},
		{"shop", "Acme..Billing", "not a C# namespace"},
	} {
		b := irtest.New(tc.pkg)
		if tc.ns != "" {
			b.Model.Targets = block(directive("namespace", tc.ns))
		}
		b.Own(value("Order", irtest.Field("id", b.Named("string"))))
		resp := generate(t, b.Model)
		if len(resp.GetFiles()) != 0 || len(resp.GetDiagnostics()) != 1 ||
			resp.GetDiagnostics()[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR ||
			!strings.Contains(resp.GetDiagnostics()[0].GetMessage(), tc.want) {
			t.Errorf("package %q, namespace %q: files = %d, diagnostics = %+v", tc.pkg, tc.ns, len(resp.GetFiles()), resp.GetDiagnostics())
		}
	}
}

func TestNames(t *testing.T) {
	b := irtest.New("shop")
	b.Own(named(value("Account", irtest.Field("event", b.Named("string"))), directive("name", "Customer")))
	b.Model.Decls[len(b.Model.Decls)-1].GetStructure().Fields[0].Directives = []*ir.Directive{directive("name", "event")}
	b.Own(value("Card", irtest.Field("number", b.Named("string"))))
	// Inside Payment, the variant Card shadows the record Card.
	b.Own(enum("Payment", variant("Card", irtest.Field("card", b.Named("Card")), irtest.Field("other", b.Named("Account")))))
	b.Own(value("Holder", irtest.Field("customer", b.Named("Account"))))

	resp := generate(t, b.Model)
	if !warned(resp, "Payment.Card.card would be the property Card") {
		t.Errorf("a property named like its variant does not warn: %+v", resp.GetDiagnostics())
	}

	b.Model.Decls[len(b.Model.Decls)-2].GetEnumeration().Variants[0].Fields[0].Meta.Name = "info"
	resp = generate(t, b.Model)
	quiet(t, resp)
	build(t, resp)
	contains(t, file(t, resp, "Customer.g.cs"), "public sealed partial record Customer\n", "    public required string @event { get; init; }\n")
	contains(t, file(t, resp, "Payment.g.cs"),
		"        public required global::Shop.Card Info { get; init; }\n",
		"        public required Customer Other { get; init; }\n")
	contains(t, file(t, resp, "Holder.g.cs"), "    public required Customer Customer { get; init; }\n")
}

func TestCollisions(t *testing.T) {
	for _, tc := range []struct {
		name string
		decl func(b *irtest.Builder) *ir.Decl
		want string
	}{
		{"enclosing type", func(b *irtest.Builder) *ir.Decl {
			return value("Tag", irtest.Field("tag", b.Named("string")))
		}, "which a member of Tag cannot be named"},
		{"same once upper-cased", func(b *irtest.Builder) *ir.Decl {
			return value("Pair", irtest.Field("id", b.Named("string")), irtest.Field("Id", b.Named("string")))
		}, "would both be the property Id"},
		{"record member", func(b *irtest.Builder) *ir.Decl {
			return value("Thing", irtest.Field("equalityContract", b.Named("string")))
		}, "which every record declares"},
		{"variant member", func(b *irtest.Builder) *ir.Decl {
			return enum("Kind", variant("ToString"))
		}, "which every type declares"},
		{"variant like enum", func(b *irtest.Builder) *ir.Decl {
			return enum("Kind", variant("Kind"))
		}, "which a member of Kind cannot be named"},
		{"equality class", func(b *irtest.Builder) *ir.Decl {
			return value("TdlEquality")
		}, "which the generated equality class takes"},
		{"newtype named Value", func(b *irtest.Builder) *ir.Decl {
			return newtype("Value", b.Named("string"))
		}, "which its own property takes"},
		{"not an identifier", func(b *irtest.Builder) *ir.Decl {
			return named(value("Thing"), directive("name", "Two Words"))
		}, "is not an identifier"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := irtest.New("shop")
			b.Own(tc.decl(b))
			b.Own(value("Fine"))
			resp := generate(t, b.Model)
			if !warned(resp, tc.want) {
				t.Errorf("no warning containing %q: %+v", tc.want, resp.GetDiagnostics())
			}
			if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "Fine.g.cs" {
				t.Errorf("the colliding declaration is emitted: %s", dump(resp))
			}
		})
	}

	b := irtest.New("shop")
	b.Own(value("Order"))
	b.Own(named(value("Purchase"), directive("name", "Order")))
	resp := generate(t, b.Model)
	if !warned(resp, "Purchase would be named Order in C#, and Order already is") {
		t.Errorf("two declarations named alike do not warn: %+v", resp.GetDiagnostics())
	}
}

func TestSkipped(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Blobs", irtest.Field("seen", b.Named("Set", b.Named("bytes")))))
	b.Own(value("Grid", irtest.Field("cells", b.Named("Map", b.Named("List", b.Named("int")), b.Named("string")))))
	b.Own(value("Holder", irtest.Field("blobs", b.Named("Blobs"))))
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Page"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: irtest.Params("T"), Fields: []*ir.Field{irtest.Field("items", b.Param("T", 0))}}},
	})
	b.Own(value("Remote", irtest.Field("x", b.Extern("other.Thing"))))
	b.Own(value("Fine", irtest.Field("x", b.Named("string"))))

	resp := generate(t, b.Model)
	for _, want := range []string{
		"a set's element cannot be bytes",
		"a map's key cannot be a collection",
		"Holder names Blobs",
		"Page is parameterized",
		"other.Thing is declared in another package",
	} {
		if !warned(resp, want) {
			t.Errorf("no warning containing %q: %+v", want, resp.GetDiagnostics())
		}
	}
	if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "Fine.g.cs" {
		t.Errorf("a skipped declaration is emitted: %s", dump(resp))
	}
}

func TestFieldDirectiveOnDeclaration(t *testing.T) {
	b := irtest.New("shop")
	b.Own(named(value("Order"), directive("file", "Order.cs")))
	resp := generate(t, b.Model)
	if !warned(resp, "belongs in the target block") {
		t.Errorf("file on a declaration does not warn: %+v", resp.GetDiagnostics())
	}
}

// TestEquality is the plan's test: two orders holding equal item lists
// compare equal, and so does every collection the record holds.
func TestEquality(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("LineItem", irtest.Field("sku", b.Named("string"))))
	b.Own(newtype("Skus", b.Named("List", b.Named("string"))))
	b.Own(value("Order",
		irtest.Field("id", b.Named("string")),
		irtest.Field("items", b.Named("List", b.Named("LineItem"))),
		irtest.Field("tags", b.Named("Set", b.Named("string"))),
		irtest.Field("nested", b.Named("Map", b.Named("string"), b.Named("List", b.Named("Option", b.Named("bytes"))))),
		irtest.Field("receipt", b.Named("bytes")),
		irtest.Field("skus", b.Named("Skus")),
		irtest.Field("note", b.Named("Option", b.Named("List", b.Named("int")))),
	))
	b.Own(enum("Payment", variant("Split", irtest.Field("parts", b.Named("List", b.Named("int"))))))

	resp := generate(t, b.Model)
	quiet(t, resp)
	out := run(t, resp, `using System.Collections.Immutable;
using Shop;

static Order Make(string id) => new Order
{
    Id = id,
    Items = ImmutableList.Create(new LineItem { Sku = "a" }, new LineItem { Sku = "b" }),
    Tags = ImmutableHashSet.Create("x", "y"),
    Nested = ImmutableDictionary<string, IImmutableList<byte[]?>>.Empty.Add("k", ImmutableList.Create<byte[]?>(new byte[] { 1, 2 }, null)),
    Receipt = new byte[] { 3 },
    Skus = new Skus(ImmutableList.Create("s")),
};

var a = Make("1");
var b = Make("1");
System.Console.WriteLine(string.Join(" ", new[]
{
    a == b,
    a.GetHashCode() == b.GetHashCode(),
    a != Make("2"),
    a != (b with { Receipt = new byte[] { 4 } }),
    a == (b with { Tags = ImmutableHashSet.Create("y", "x") }),
    new Payment.Split { Parts = ImmutableList.Create(1L) } == new Payment.Split { Parts = ImmutableList.Create(1L) },
    new Skus(ImmutableList.Create("s")) == new Skus(ImmutableList.Create("s")),
    // A comparer of the consumer's own changes neither equality nor the hash.
    Folded(a) != a && a != Folded(a),
    Folded(a).GetHashCode() == Folded(b).GetHashCode(),
}));

static Order Folded(Order o) => o with
{
    Tags = ImmutableHashSet.Create(System.StringComparer.OrdinalIgnoreCase, "X", "x", "y"),
    Nested = ImmutableDictionary.Create<string, IImmutableList<byte[]?>>(System.StringComparer.OrdinalIgnoreCase).Add("K", ImmutableList.Create<byte[]?>()),
};
`)
	if strings.TrimSpace(out) != "True True True True True True True True True" {
		t.Errorf("equality = %q", out)
	}
}

// TestSmoke builds the smoke model, which checks.gen-csharp builds too.
func TestSmoke(t *testing.T) {
	m := irtest.Conformance(t, "../../testdata/gen/smoke")
	resp := generate(t, m)
	for _, d := range resp.GetDiagnostics() {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
			t.Errorf("error: %v", d)
		}
	}
	build(t, resp)
}

// TestConformance generates every conformance case and builds what it
// generates.
func TestConformance(t *testing.T) {
	dirs, err := filepath.Glob("../../testdata/conformance/*")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, "pending")); err == nil {
			continue
		}
		t.Run(filepath.Base(dir), func(t *testing.T) {
			t.Parallel()
			m := irtest.Conformance(t, dir)
			resp := generate(t, m)
			for _, d := range resp.GetDiagnostics() {
				if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("error: %v", d)
				}
			}
			if len(resp.GetFiles()) > 0 {
				build(t, resp)
			}
		})
	}
}
