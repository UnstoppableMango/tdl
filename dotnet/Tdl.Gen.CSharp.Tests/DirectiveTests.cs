using static Tdl.Gen.CSharp.Tests.Check;

namespace Tdl.Gen.CSharp.Tests;

// The directives of phase 2 of csharp-backend-plan.md: key, defaults,
// foreign, attribute, and references into a dependency.
public class DirectiveTests
{
	[Fact]
	public void Keys()
	{
		var resp = Generate("keys/source");
		Quiet(resp);
		Contains(File(resp, "User.g.cs"), "    public required string Name { get; init; }\n\n    public UserId Key() => Id;\n}\n");
		Contains(File(resp, "LineItem.g.cs"),
			"\n    public LineItemKey Key() => new(Order, Sku, Kind);\n\n    public bool Equals(LineItem? other) =>",
			"}\n\n/// <summary>The fields identifying a <see cref=\"LineItem\"/>.</summary>\npublic readonly partial record struct LineItemKey(string Order, Sku Sku, string Kind);\n");
		var output = Run(resp, """
			using System.Collections.Immutable;
			using Shop;

			#pragma warning disable CS0618
			public static class Program
			{
			    public static string Run()
			    {
			        var a = new LineItem { Order = "o", Sku = new Sku("s"), Kind = "t", Tags = ImmutableList.Create("x") };
			        var b = a with { Tags = ImmutableList.Create("y") };
			        var byKey = new System.Collections.Generic.Dictionary<LineItemKey, LineItem> { [a.Key()] = a };
			        return string.Join(" ", new[]
			        {
			            a.Key() == b.Key(),
			            byKey.ContainsKey(b.Key()),
			            a.Key() != (a with { Sku = new Sku("t") }).Key(),
			            new User { Id = new UserId("u"), Name = "n" }.Key() == new UserId("u"),
			        });
			    }
			}
			""");
		Assert.Equal("True True True True", output);
	}

	[Theory]
	[InlineData("keys/value", "only an entity is identified by one")]
	[InlineData("keys/enum", "only an entity is identified by one")]
	[InlineData("keys/missing", "Order has no field code to key on")]
	[InlineData("keys/twice", "names id twice")]
	[InlineData("keys/string", "a key names fields")]
	[InlineData("keys/collection", "compares a collection or bytes by reference")]
	[InlineData("keys/bytes", "compares a collection or bytes by reference")]
	[InlineData("keys/property", "has a property Key")]
	[InlineData("keys/record", "the key record OrderKey would collide with OrderKey")]
	public void KeysThatCannotBeGenerated(string name, string want)
	{
		var resp = Generate(name);
		Warned(resp, want);
		Absent(File(resp, "Order.g.cs"), "Key()");
		Compile(resp);
	}

	// The plan's test: a defaulted field left out of a constructing
	// expression holds its default.
	[Fact]
	public void Defaults()
	{
		var resp = Generate("defaults/source");
		Warned(resp, "the default \"2026-01-01\" is dropped: C# has no constant of type instant");
		Contains(File(resp, "Settings.g.cs"),
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
			"    public required global::System.DateTimeOffset At { get; init; }\n");
		var output = Run(resp, """
			using Shop;

			public static class Program
			{
			    public static string Run()
			    {
			        var s = new Settings { Name = "x", At = System.DateTimeOffset.UnixEpoch };
			        return string.Join(" ", new object[]
			        {
			            s.Status, s.Payment is Payment.Cash, s.Retries, s.Small, s.Big, s.Ratio, s.Scale, s.Price, s.Enabled,
			            s.Greeting, s.Contact.Value, s.Note!, string.Join(",", System.Linq.Enumerable.Order(s.Tags)), s.Lines.Count, s.Codes[0], s.Counts.Count,
			        });
			    }
			}
			""");
		Assert.Equal("Pending True 3 -2 18446744073709551615 1.5 2 9.99 True hi \"there\" a@b.c n a,b 0 Shipped 0", output);
	}

	[Fact]
	public void DefaultsThatCannotBeWritten()
	{
		var resp = Generate("defaults/dropped");
		Warned(resp, "is dropped: instant is a foreign type");
		Warned(resp, "is dropped: it is out of range for int32");
		Warned(resp, "is dropped: it is out of range for float32");
		Warned(resp, "is dropped: it is out of range for decimal");
		Contains(File(resp, "Settings.g.cs"),
			"    public required global::NodaTime.Instant At { get; init; }\n",
			"    public required int Small { get; init; }\n",
			"    public required float Huge { get; init; }\n",
			"    public required decimal Money { get; init; }\n");
	}

	// NodaTime stands in for the NodaTime package: only the type's name
	// matters to the compiler.
	const string NodaTime = """
		namespace NodaTime;

		public readonly record struct Instant(long Ticks);
		""";

	// The plan's test: instant => foreign("NodaTime", "Instant") builds
	// against NodaTime.
	[Fact]
	public void Foreign()
	{
		var resp = Generate("foreign/source");
		Quiet(resp);
		Contains(File(resp, "Order.g.cs"),
			"    public required global::NodaTime.Instant Placed { get; init; }\n",
			"IImmutableList<global::NodaTime.Instant> History",
			"    public required global::System.Decimal Total { get; init; }\n",
			"IImmutableDictionary<string, global::System.Decimal> Totals",
			// A segment C# reserves is escaped.
			"    public global::Acme.@event.Label? Label { get; init; }\n");
		Assert.DoesNotContain("Money.g.cs", Paths(resp));
		Assert.DoesNotContain("Label.g.cs", Paths(resp));
		Compile([resp], [NodaTime, "namespace Acme.@event;\n\npublic sealed record Label;\n"]);
	}

	[Fact]
	public void ForeignThatCannotBeUsed()
	{
		var resp = Generate("foreign/unusable");
		Warned(resp, "the foreign mapping of Money names the namespace \"acme money\", which is not a .NET namespace");
		Warned(resp, "the foreign mapping of Coin names \"Coin<int>\", which is not a .NET identifier");
		Warned(resp, "Account is a foreign type, so its key is not generated");
		Warned(resp, "min(0) is not checked: Account is a foreign type");
		File(resp, "Money.g.cs");
		File(resp, "Coin.g.cs");
		Compile(resp);
	}

	[Fact]
	public void ForeignExtern()
	{
		var resp = Generate("foreign-extern/source");
		Quiet(resp);
		Contains(File(resp, "Order.g.cs"), "    public required global::Acme.Money.Amount Total { get; init; }\n");
	}

	[Fact]
	public void Attribute()
	{
		var resp = Generate("attribute/source");
		Warned(resp, "attribute names \"Not.Valid\", which is not a .NET identifier");
		Contains(File(resp, "Order.g.cs"),
			"\n[global::System.Serializable]\n[global::System.Runtime.Serialization.DataContract]\n[global::System.Diagnostics.DebuggerNonUserCode]\npublic sealed partial record Order\n");
		Contains(File(resp, "Status.g.cs"), "\n[global::System.Serializable]\n[global::System.Flags]\npublic enum Status\n");
		Contains(File(resp, "Bad.g.cs"), "\n[global::System.Serializable]\npublic sealed partial record Bad;\n");
		Compile(resp);
	}

	// The plan's test: a two-package model generated into one project
	// builds, a reference into the dependency naming the type its csharp
	// target block generates.
	[Fact]
	public void DependencyWithCSharpBlock()
	{
		var resp = Generate("dependency/source");
		Quiet(resp);
		Contains(File(resp, "Order.g.cs"),
			"    public required global::Acme.Currencies.Money Total { get; init; }\n",
			"    public required global::Acme.Currencies.Iso4217 Currency { get; init; }\n",
			"IImmutableList<global::Acme.Currencies.Rate> Rates");
		var dep = Generate("dependency/money");
		Quiet(dep);
		Compile(resp, dep);
	}

	[Fact]
	public void DependencyWithoutCSharpBlock()
	{
		var resp = Generate("extern/source");
		Warned(resp, "acme.money.Money has no C# type: give it a foreign directive, or its package a csharp target block");
		Warned(resp, "Cart names Order");
		Assert.Equal(["Kept.g.cs"], Paths(resp));
		Compile(resp);
	}
}
