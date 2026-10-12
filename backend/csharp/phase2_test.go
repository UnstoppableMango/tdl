package csharp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/csharp"
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
	d := &ir.Directive{Name: "key", Target: csharp.Name}
	for _, f := range fields {
		d.Args = append(d.Args, irtest.Name(f))
	}
	return d
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

// buildTogether builds every response as one project, each in its own
// directory since each namespace declares its own TdlEquality.g.cs, with
// extra files beside them.
func buildTogether(t *testing.T, extra map[string]string, resps ...*plugin.Response) {
	t.Helper()
	bin := dotnet(t)
	if bin == "" {
		return
	}
	dir := project(t, &plugin.Response{}, "")
	var all []string
	write := func(path, content string) {
		path = filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i, resp := range resps {
		for _, f := range resp.GetFiles() {
			path := string(rune('a'+i)) + "/" + f.GetPath()
			write(path, string(f.GetContent()))
			all = append(all, "==> "+path+" <==\n"+string(f.GetContent()))
		}
	}
	for path, content := range extra {
		write(path, content)
	}
	if out, err := command(bin, dir, "build").CombinedOutput(); err != nil {
		t.Errorf("dotnet build rejects the responses: %v\n%s\n%s", err, out, strings.Join(all, "\n"))
	}
}

func TestKeys(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("UserId", b.Named("string")))
	b.Own(named(entity("User", irtest.Field("id", b.Named("UserId")), irtest.Field("name", b.Named("string"))), keyOn("id")))
	b.Own(newtype("Sku", b.Named("string")))
	typ := irtest.Field("type", b.Named("string"))
	typ.Meta.Deprecated = &ir.Deprecation{Reason: "gone"}
	b.Own(named(entity("LineItem",
		irtest.Field("order", b.Named("string")),
		irtest.Field("sku", b.Named("Sku")),
		typ,
		irtest.Field("tags", b.Named("List", b.Named("string"))),
	), keyOn("order", "sku", "type")))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, file(t, resp, "User.g.cs"), "    public required string Name { get; init; }\n\n    public UserId Key() => Id;\n}\n")
	contains(t, file(t, resp, "LineItem.g.cs"),
		"\n    public LineItemKey Key() => new(Order, Sku, Type);\n\n    public bool Equals(LineItem? other) =>",
		"}\n\n/// <summary>The fields identifying a <see cref=\"LineItem\"/>.</summary>\npublic readonly partial record struct LineItemKey(string Order, Sku Sku, string Type);\n",
	)
	out := run(t, resp, `using System.Collections.Immutable;
using Shop;

#pragma warning disable CS0618
var a = new LineItem { Order = "o", Sku = new Sku("s"), Type = "t", Tags = ImmutableList.Create("x") };
var b = a with { Tags = ImmutableList.Create("y") };
var byKey = new System.Collections.Generic.Dictionary<LineItemKey, LineItem> { [a.Key()] = a };
System.Console.WriteLine(string.Join(" ", new[]
{
    a.Key() == b.Key(),
    byKey.ContainsKey(b.Key()),
    a.Key() != (a with { Sku = new Sku("t") }).Key(),
    new User { Id = new UserId("u"), Name = "n" }.Key() == new UserId("u"),
}));
`)
	if strings.TrimSpace(out) != "True True True True" {
		t.Errorf("keys = %q", out)
	}
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
				return named(value("Order", irtest.Field("id", b.Named("string"))), keyOn("id"))
			},
			want: "only an entity is identified by one",
		},
		{
			name: "enum",
			decl: func(*irtest.Builder) *ir.Decl {
				return named(enum("Order", variant("A")), keyOn("A"))
			},
			want: "only an entity is identified by one",
		},
		{
			name: "missing field",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("id", b.Named("string"))), keyOn("code"))
			},
			want: "Order has no field code to key on",
		},
		{
			name: "twice",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("id", b.Named("string"))), keyOn("id", "id"))
			},
			want: "names id twice",
		},
		{
			name: "string",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("id", b.Named("string"))), directive("key", "id"))
			},
			want: "a key names fields",
		},
		{
			name: "collection",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("lines", b.Named("List", b.Named("string")))), keyOn("lines"))
			},
			want: "compares a collection or bytes by reference",
		},
		{
			name: "bytes",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("hash", b.Named("bytes"))), keyOn("hash"))
			},
			want: "compares a collection or bytes by reference",
		},
		{
			name: "property named Key",
			decl: func(b *irtest.Builder) *ir.Decl {
				return named(entity("Order", irtest.Field("key", b.Named("string"))), keyOn("key"))
			},
			want: "has a property Key",
		},
		{
			name: "record collision",
			decl: func(b *irtest.Builder) *ir.Decl {
				b.Own(value("OrderKey"))
				return named(entity("Order", irtest.Field("a", b.Named("string")), irtest.Field("b", b.Named("string"))), keyOn("a", "b"))
			},
			want: "the key record OrderKey would collide with OrderKey",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := irtest.New("shop")
			b.Own(tc.decl(b))
			resp := generate(t, b.Model)
			if !warned(resp, tc.want) {
				t.Errorf("no warning %q: %+v", tc.want, resp.GetDiagnostics())
			}
			src := file(t, resp, "Order.g.cs")
			absent(t, src, "Key()")
			build(t, resp)
		})
	}
}

const defaultsSource = `package shop

type Email: string

enum Status { Pending Shipped }

enum Payment {
  Cash
  Card { last4: string }
}

type Settings {
  name: string
  status: Status = Pending
  payment: Payment = Cash
  retries: int = 3
  small: int32 = -2
  big: uint64 = 18446744073709551615
  ratio: float32 = 1.5
  scale: float64 = 2
  price: decimal = 9.99
  enabled: bool = true
  greeting: string = "hi \"there\""
  contact: Email = "a@b.c"
  note: string? = "n"
  tags: {string} = ["a", "b"]
  lines: [int] = []
  codes: [Status] = [Shipped]
  counts: {string -> int} = []
  at: instant = "2026-01-01"
}

target csharp for shop { }
`

// TestDefaults is the plan's test: a defaulted field left out of a
// constructing expression holds its default.
func TestDefaults(t *testing.T) {
	resp := generate(t, lowerSource(t, "shop.tdl", defaultsSource, nil))
	if !warned(resp, `the default "2026-01-01" is dropped: C# has no constant of type instant`) {
		t.Errorf("a default on instant does not warn: %+v", resp.GetDiagnostics())
	}
	contains(t, file(t, resp, "Settings.g.cs"),
		"    public required string Name { get; init; }\n",
		"    public Status Status { get; init; } = Status.Pending;\n",
		"    public Payment Payment { get; init; } = new Payment.Cash();\n",
		"    public long Retries { get; init; } = 3;\n",
		"    public int Small { get; init; } = -2;\n",
		"    public ulong Big { get; init; } = 18446744073709551615;\n",
		"    public float Ratio { get; init; } = 1.5f;\n",
		"    public double Scale { get; init; } = 2.0;\n",
		"    public decimal Price { get; init; } = 9.99m;\n",
		"    public bool Enabled { get; init; } = true;\n",
		"    public string Greeting { get; init; } = \"hi \\\"there\\\"\";\n",
		"    public Email Contact { get; init; } = new Email(\"a@b.c\");\n",
		"    public string? Note { get; init; } = \"n\";\n",
		"IImmutableSet<string> Tags { get; init; } = global::System.Collections.Immutable.ImmutableHashSet.Create<string>(\"a\", \"b\");\n",
		"IImmutableList<long> Lines { get; init; } = global::System.Collections.Immutable.ImmutableList<long>.Empty;\n",
		"IImmutableList<Status> Codes { get; init; } = global::System.Collections.Immutable.ImmutableList.Create<Status>(Status.Shipped);\n",
		"IImmutableDictionary<string, long> Counts { get; init; } = global::System.Collections.Immutable.ImmutableDictionary<string, long>.Empty;\n",
		"    public required global::System.DateTimeOffset At { get; init; }\n",
	)
	out := run(t, resp, `using Shop;

var s = new Settings { Name = "x", At = System.DateTimeOffset.UnixEpoch };
System.Console.WriteLine(string.Join(" ", new object[]
{
    s.Status, s.Payment is Payment.Cash, s.Retries, s.Small, s.Big, s.Ratio, s.Scale, s.Price, s.Enabled,
    s.Greeting, s.Contact.Value, s.Note!, string.Join(",", System.Linq.Enumerable.Order(s.Tags)), s.Lines.Count, s.Codes[0], s.Counts.Count,
}));
`)
	want := `Pending True 3 -2 18446744073709551615 1.5 2 9.99 True hi "there" a@b.c n a,b 0 Shipped 0`
	if strings.TrimSpace(out) != want {
		t.Errorf("defaults = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestDefaultsThatCannotBeWritten(t *testing.T) {
	b := irtest.New("shop")
	_, id, _ := b.Model.FindDecl("instant")
	b.Model.Decls[id.GetIndex()].Directives = []*ir.Directive{directive("foreign", "NodaTime", "Instant")}
	at := irtest.Field("at", b.Named("instant"))
	at.DefaultValue = irtest.Text("2026-01-01")
	small := irtest.Field("small", b.Named("int32"))
	small.DefaultValue = &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "4294967296"}
	b.Own(value("Settings", at, small))

	resp := generate(t, b.Model)
	for _, want := range []string{
		"is dropped: instant is a foreign type",
		"is dropped: it is out of range for int32",
	} {
		if !warned(resp, want) {
			t.Errorf("no warning %q: %+v", want, resp.GetDiagnostics())
		}
	}
	contains(t, file(t, resp, "Settings.g.cs"),
		"    public required global::NodaTime.Instant At { get; init; }\n",
		"    public required int Small { get; init; }\n",
	)
}

// nodaTime stands in for the NodaTime package, which the test project
// cannot restore offline: only the type's name matters to the build.
const nodaTime = `namespace NodaTime;

public readonly record struct Instant(long Ticks);
`

// TestForeign is the plan's test: instant => foreign("NodaTime", "Instant")
// builds against NodaTime.
func TestForeign(t *testing.T) {
	b := irtest.New("shop")
	// A primitive mapped by a target path, as `instant => foreign(...)` is.
	_, id, _ := b.Model.FindDecl("instant")
	b.Model.Decls[id.GetIndex()].Directives = []*ir.Directive{directive("foreign", "NodaTime", "Instant")}
	b.Own(named(value("Money", irtest.Field("units", b.Named("int"))), directive("foreign", "System", "Decimal")))
	// A segment C# reserves is escaped.
	b.Own(named(value("Label"), directive("foreign", "Acme.event", "Label")))
	b.Own(value("Order",
		irtest.Field("placed", b.Named("instant")),
		irtest.Field("history", b.Named("List", b.Named("instant"))),
		irtest.Field("total", b.Named("Money")),
		irtest.Field("totals", b.Named("Map", b.Named("string"), b.Named("Money"))),
		irtest.Field("label", b.Named("Option", b.Named("Label"))),
	))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, file(t, resp, "Order.g.cs"),
		"    public required global::NodaTime.Instant Placed { get; init; }\n",
		"IImmutableList<global::NodaTime.Instant> History",
		"    public required global::System.Decimal Total { get; init; }\n",
		"IImmutableDictionary<string, global::System.Decimal> Totals",
		"    public global::Acme.@event.Label? Label { get; init; }\n",
	)
	for _, f := range resp.GetFiles() {
		if f.GetPath() == "Money.g.cs" || f.GetPath() == "Label.g.cs" {
			t.Errorf("a foreign declaration is generated: %s", f.GetPath())
		}
	}
	buildTogether(t, map[string]string{
		"NodaTime.cs": nodaTime,
		"Label.cs":    "#pragma warning disable CA1716\nnamespace Acme.@event;\n\npublic sealed record Label;\n",
	}, resp)
}

func TestForeignThatCannotBeUsed(t *testing.T) {
	b := irtest.New("shop")
	b.Own(named(value("Money", irtest.Field("units", b.Named("int"))), directive("foreign", "acme money", "Money")))
	b.Own(named(value("Coin"), directive("foreign", "Acme", "Coin<int>")))
	q := irtest.Field("units", b.Named("int"))
	q.Constraints = []*ir.Constraint{{Name: "min", Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "0"}}}}
	b.Own(named(entity("Account", irtest.Field("id", b.Named("string")), q),
		directive("foreign", "Acme", "Account"), keyOn("id")))

	resp := generate(t, b.Model)
	for _, want := range []string{
		`the foreign mapping of Money names the namespace "acme money", which is not a .NET namespace`,
		`the foreign mapping of Coin names "Coin<int>", which is not a .NET identifier`,
		"Account is a foreign type, so its key is not generated",
		"min(0) is not checked: Account is a foreign type",
	} {
		if !warned(resp, want) {
			t.Errorf("no warning %q: %+v", want, resp.GetDiagnostics())
		}
	}
	file(t, resp, "Money.g.cs")
	file(t, resp, "Coin.g.cs")
	build(t, resp)
}

func TestForeignExtern(t *testing.T) {
	b := irtest.New("shop")
	money := b.ExternIn("acme.money", "Money", directive("foreign", "Acme.Money", "Amount"))
	b.Own(value("Order", irtest.Field("total", money)))

	resp := generate(t, b.Model)
	quiet(t, resp)
	contains(t, file(t, resp, "Order.g.cs"), "    public required global::Acme.Money.Amount Total { get; init; }\n")
}

func TestAttribute(t *testing.T) {
	b := irtest.New("shop")
	b.Model.Targets = block(directive("attribute", "System", "Serializable"))
	b.Own(named(value("Order", irtest.Field("id", b.Named("string"))),
		directive("attribute", "System.Runtime.Serialization", "DataContract"),
		directive("attribute", "System.Diagnostics", "DebuggerNonUserCode"),
	))
	b.Own(named(enum("Status", variant("Pending")), directive("attribute", "System", "Flags")))
	b.Own(named(value("Bad"), directive("attribute", "System", "Not.Valid")))

	resp := generate(t, b.Model)
	if !warned(resp, `attribute names "Not.Valid", which is not a .NET identifier`) {
		t.Errorf("an unusable attribute does not warn: %+v", resp.GetDiagnostics())
	}
	contains(t, file(t, resp, "Order.g.cs"),
		"\n[global::System.Serializable]\n[global::System.Runtime.Serialization.DataContract]\n[global::System.Diagnostics.DebuggerNonUserCode]\npublic sealed partial record Order\n",
	)
	contains(t, file(t, resp, "Status.g.cs"), "\n[global::System.Serializable]\n[global::System.Flags]\npublic enum Status\n")
	contains(t, file(t, resp, "Bad.g.cs"), "\n[global::System.Serializable]\npublic sealed partial record Bad;\n")
	build(t, resp)
}

const moneySource = `package acme.money

type Money { units: int }

type Currency { code: string }

type Rate: decimal

target csharp for acme.money {
  namespace("Acme.Currencies")
  Currency => name("Iso4217")
}
`

// TestDependencyWithCsharpBlock is the plan's test: a two-package model
// generated into one project builds, a reference into the dependency naming
// the type its csharp target block generates.
func TestDependencyWithCsharpBlock(t *testing.T) {
	d := deps{"money.tdl": moneySource}
	root := lowerSource(t, "shop.tdl", `package shop

import "money.tdl" as money

type Order {
  total: money.Money
  currency: money.Currency
  rates: [money.Rate]
}

target csharp for shop { }
`, d)
	resp := generate(t, root)
	quiet(t, resp)
	contains(t, file(t, resp, "Order.g.cs"),
		"    public required global::Acme.Currencies.Money Total { get; init; }\n",
		"    public required global::Acme.Currencies.Iso4217 Currency { get; init; }\n",
		"IImmutableList<global::Acme.Currencies.Rate> Rates",
	)
	dep := generate(t, lowerSource(t, "money.tdl", moneySource, nil))
	quiet(t, dep)
	buildTogether(t, nil, resp, dep)
}

func TestDependencyWithoutCsharpBlock(t *testing.T) {
	d := deps{"money.tdl": "package acme.money\n\ntype Money { units: int }\n"}
	root := lowerSource(t, "shop.tdl", `package shop

import "money.tdl" as money

type Order { total: money.Money }

type Cart { order: Order }

type Kept { id: string }
`, d)
	resp := generate(t, root)
	if !warned(resp, "acme.money.Money has no C# type: give it a foreign directive, or its package a csharp target block") ||
		!warned(resp, "Cart names Order") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	file(t, resp, "Kept.g.cs")
	for _, f := range resp.GetFiles() {
		if f.GetPath() != "Kept.g.cs" {
			t.Errorf("a declaration naming an unmapped extern is generated: %s", f.GetPath())
		}
	}
	build(t, resp)
}
