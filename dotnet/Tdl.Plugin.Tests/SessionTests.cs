using Tdl.Ir.V1;
using Tdl.Plugin.V1;

namespace Tdl.Plugin.Tests;

public class SessionTests
{
	static Session Catalog() => new(Repo.Request("catalog"), "Debug");

	static Decl Named(Session s, string name) => s.Model.Decls.Single(d => d.Meta.Name == name);

	[Fact]
	public void RefusesARequestWithoutAnAnalysis()
	{
		var req = Repo.Request("shop");
		req.Analysis = null;

		Assert.Throws<ArgumentException>(() => new Session(req, "Debug"));
	}

	[Fact]
	public void RefusesAnAnalysisOfAnotherModel()
	{
		var req = Repo.Request("shop");
		req.Analysis.Decls.RemoveAt(0);

		Assert.Throws<ArgumentException>(() => new Session(req, "Debug"));
	}

	[Fact]
	public void OwnLeavesOutThePrelude()
	{
		var s = Catalog();

		Assert.Equal(["Product", "Marker", "Availability"], s.Own().Select(d => d.Meta.Name));
	}

	[Fact]
	public void NamesFromTheShippedWords()
	{
		var s = Catalog();
		var product = Named(s, "Product");
		var availability = Named(s, "Availability");

		Assert.Equal("product", s.DeclName(product, Case.Snake));
		Assert.Equal("ID", s.FieldName(product.Structure.Fields[0], Case.ScreamingSnake));
		Assert.Equal("in_stock", s.VariantName(availability.Enumeration.Variants[0], Case.Snake));
	}

	[Fact]
	public void ANameDirectiveWins()
	{
		var s = Catalog();
		var product = Named(s, "Product");
		product.Directives.Add(new Directive { Name = "name", Target = "debug", Args = { new Literal { Text = "Item" } } });
		product.Directives.Add(new Directive { Name = "name", Target = "other", Args = { new Literal { Text = "Other" } } });

		Assert.Equal("Item", s.DeclName(product, Case.Snake));
	}

	[Fact]
	public void ReadsTheTargetBlock()
	{
		var s = Catalog();

		Assert.Equal("./gen", s.Block("out")?.Args[0].Text);
		Assert.Null(s.Block("note"));
		Assert.True(s.Bare("roundtrip"));
		Assert.False(s.Bare("closed"));
	}

	[Fact]
	public void WarnsWithAPositionAndACode()
	{
		var s = Catalog();
		var pos = new Position { Filename = "a.tdl", Line = 1 };

		s.Warn(new Unsupported("lost", pos, LossCodes.Collection));
		s.Warn(new Invalid("wrong", pos));
		s.Error(null, "broken");
		s.Lossy(LossCodes.Doc, pos, "undocumented");

		Assert.Equal(
			[
				(Severity.Warning, "lost", LossCodes.Collection),
				(Severity.Error, "wrong", ""),
				(Severity.Error, "broken", ""),
				(Severity.Warning, "undocumented", LossCodes.Doc),
			],
			s.Diagnostics.Select(d => (d.Severity, d.Message, d.Code)));
		Assert.Equal(pos, s.Diagnostics[0].Position);
		Assert.Equal(2, s.Response([]).Diagnostics.Count(d => d.Severity == Severity.Error));
	}

	[Fact]
	public void WarnsAboutUnenforcedConstraints()
	{
		var s = Catalog();

		s.WarnConstraints(Named(s, "Product"));
		s.WarnConstraints(Named(s, "Marker"));

		var d = Assert.Single(s.Diagnostics);
		Assert.Equal("Product.name carries 1 constraint(s), and validation is not generated yet", d.Message);
		Assert.Equal(LossCodes.Constraint, d.Code);
	}

	[Fact]
	public void AnEnforcedConstraintDoesNotWarn()
	{
		var s = Catalog();

		s.WarnConstraints(Named(s, "Product"), c => c.Name == "length");

		Assert.Empty(s.Diagnostics);
	}

	[Fact]
	public void SaysWhatADeclarationDeclares()
	{
		var s = Catalog();

		Assert.True(Session.Declares(Named(s, "Product"), out var problem));
		Assert.Null(problem);
		Assert.False(Session.Declares(Named(s, "string"), out problem));
		Assert.Null(problem);
		Assert.False(Session.Declares(Named(s, "Entity"), out problem));
		Assert.Equal("Entity is a class, and classes are not generated yet", problem?.Message);
		Assert.False(Session.Declares(Named(s, "Option"), out problem));
		Assert.IsType<Unsupported>(problem);
	}

	[Fact]
	public void FollowsReferences()
	{
		var s = Cascading(out var a, out var b, out var c, out _);

		Assert.Equal([b], s.References(a));
		Assert.Equal([c], s.References(b));
		Assert.Equal([b, c], s.TypeReferences(new ID { Index = 0 }, new ID { Index = 1 }, new ID { Index = 0 }, new ID { Index = 9 }));
	}

	[Fact]
	public void CascadesSkips()
	{
		var s = Cascading(out var a, out var b, out var c, out var d);
		var skipped = new HashSet<Decl>(ReferenceEqualityComparer.Instance) { c };

		s.Cascade([a, b, c, d], skipped);

		Assert.Equal([c, b, a], skipped);
		Assert.Equal(
			["B names C, which is not generated", "A names B, which is not generated"],
			s.Diagnostics.Select(x => x.Message));
	}

	// Cascading is A naming B naming C, and D naming nothing, with the
	// analysis tdl would send.
	static Session Cascading(out Decl a, out Decl b, out Decl c, out Decl d)
	{
		Decl Struct(string name) => new() { Meta = new Meta { Name = name }, Structure = new Struct() };
		a = Struct("A");
		b = Struct("B");
		c = Struct("C");
		d = Struct("D");

		var model = new Model { Decls = { a, b, c, d }, Types_ = { new Ir.V1.Type(), new Ir.V1.Type() } };
		var analysis = new Analysis
		{
			Types_ =
			{
				new TypeView { References = { new ID { Index = 1, Name = "B" } } },
				new TypeView { References = { new ID { Index = 2, Name = "C" } } },
			},
			Decls =
			{
				new DeclView { Own = true, References = { new ID { Index = 1, Name = "B" } } },
				new DeclView { Own = true, References = { new ID { Index = 2, Name = "C" } } },
				new DeclView { Own = true },
				new DeclView { Own = true },
			},
		};
		return new Session(new Request { Target = "x", Model = model, Analysis = analysis }, "X");
	}
}

public class RoundtripTests
{
	static Session Catalog() => new(Repo.Request("catalog"), "Debug");

	static Decl Named(Session s, string name) => s.Model.Decls.Single(d => d.Meta.Name == name);

	[Fact]
	public void ReadsTheCarriedSource()
	{
		var s = Catalog();

		Assert.Equal(["Product", "Marker", "Availability", ""], s.Items.Select(i => i.Name));
		Assert.Equal(1, s.IndexOf("Marker"));
		Assert.Equal(-1, s.IndexOf("Missing"));
		Assert.Equal("type Marker { }", s.DeclSource("Marker"));
		Assert.Equal("", s.DeclSource("Missing"));
		Assert.Equal("Entity", s.Conforms("Product"));
		Assert.Empty(s.Imports);
	}

	[Fact]
	public void ReadsAFieldsSource()
	{
		var s = Catalog();
		var category = Named(s, "Product").Structure.Fields[2];
		var until = Named(s, "Availability").Enumeration.Variants[1].Fields[0];

		Assert.Equal("category: string?", s.FieldSource(category, false));
		Assert.Equal("deprecated(\"use tags\") category: string?", s.FieldSource(category, true));
		Assert.Equal("until: string", s.FieldSource(until, true));
	}

	[Fact]
	public void CarriesWhatIsNotEmitted()
	{
		var s = Catalog();

		var carried = s.Carried(new HashSet<string> { "Product", "Availability" });

		Assert.Equal([1, 3], carried.Select(c => c.At));
		Assert.Equal("type Marker { }", carried[0].Source);
		Assert.StartsWith("target debug for catalog {", carried[1].Source);
		Assert.DoesNotContain("roundtrip", carried[1].Source);
	}

	[Fact]
	public void WithoutRoundtripThereIsNoSource()
	{
		var s = new Session(Repo.Request("shop"), "Debug");

		Assert.Empty(s.Items);
		Assert.Equal("", s.DeclSource("Order"));
		Assert.Equal("", s.FieldSource(Named(s, "Order").Structure.Fields[0], true));
	}

	[Fact]
	public void LoseIsMutedUnderRoundtrip()
	{
		var s = Catalog();
		s.Lose(LossCodes.Doc, null, "lost");
		s.Skip(new Unsupported("skipped", null));
		s.Roundtrip = true;
		s.Lose(LossCodes.Doc, null, "carried");
		s.Skip(new Unsupported("carried", null));
		s.Skip(new Invalid("wrong", null));

		Assert.Equal(["lost", "skipped", "wrong"], s.Diagnostics.Select(d => d.Message));
	}

	[Fact]
	public void LosesAStructsKind()
	{
		var s = Catalog();
		var product = Named(s, "Product");

		Assert.Equal("entity", s.LoseKind(product));
		product.Structure.Kind = StructKind.Mixin;
		Assert.Equal("mixin", s.LoseKind(product));
		Assert.Equal("", s.LoseKind(Named(s, "Marker")));
		Assert.Equal("", s.LoseKind(Named(s, "Availability")));

		Assert.Equal(
			["Product is an entity, and reads back as a value", "Product is a mixin, and reads back as a value"],
			s.Diagnostics.Select(d => d.Message));
		Assert.All(s.Diagnostics, d => Assert.Equal(LossCodes.StructKind, d.Code));
	}

	[Fact]
	public void LosesConformancesButAStructsEntity()
	{
		var s = Catalog();
		var product = Named(s, "Product");
		var availability = Named(s, "Availability");

		Assert.False(s.LoseConforms(product, "X"));
		product.Structure.Conforms.Add(new ClassRef { Class = new ID { Name = "Audited" } });
		Assert.True(s.LoseConforms(product, "X"));
		availability.Enumeration.Conforms.Add(new ClassRef { Class = new ID { Name = "Entity" } });
		Assert.True(s.LoseConforms(availability, "X"));

		Assert.Equal(
			["Product conforms to Audited, which X does not carry", "Availability conforms to Entity, which X does not carry"],
			s.Diagnostics.Select(d => d.Message));
	}

	[Fact]
	public void DeclaresLossy()
	{
		var s = Catalog();
		var aliased = new List<string>();
		var alias = new Decl { Meta = new Meta { Name = "Name" }, Alias = new Alias { Target = new ID { Name = "string" } } };

		Assert.True(s.DeclaresLossy(Named(s, "Product"), t => aliased.Add(t.Name), out var problem));
		Assert.Null(problem);
		Assert.False(s.DeclaresLossy(Named(s, "Entity"), t => aliased.Add(t.Name), out problem));
		Assert.Equal(LossCodes.Class, (problem as Unsupported)?.Code);
		Assert.False(s.DeclaresLossy(Named(s, "Option"), t => aliased.Add(t.Name), out problem));
		Assert.Equal(LossCodes.Generic, (problem as Unsupported)?.Code);
		Assert.False(s.DeclaresLossy(Named(s, "string"), t => aliased.Add(t.Name), out problem));
		Assert.Null(problem);
		Assert.False(s.DeclaresLossy(alias, t => aliased.Add(t.Name), out problem));
		Assert.Null(problem);

		Assert.Equal(["string"], aliased);
		Assert.Equal(LossCodes.Alias, Assert.Single(s.Diagnostics).Code);
	}

	[Fact]
	public void ChecksRenames()
	{
		var s = Catalog();
		var renamed = new Directive { Name = "name", Target = "debug", Args = { new Literal { Text = "x" } } };

		s.CheckRenamed([], false, "Product");
		s.CheckRenamed([renamed], true, "Product");
		s.CheckRenamed([renamed], false, "Product");
		s.CheckRenamed([], true, "Order");

		Assert.Equal(
			["Product's name directive restates the convention, and does not read back", "Order reads back with a name directive"],
			s.Diagnostics.Select(d => d.Message));
	}
}
