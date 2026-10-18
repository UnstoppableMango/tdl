using Tdl.Ir.V1;

namespace Tdl.Plugin.Tests;

public class ResolveTests
{
	static Session Catalog() => new(Repo.Request("catalog"), "Debug");

	static Field Field(Session s, string decl, string name) =>
		Session.Fields(s.Model.Decls.Single(d => d.Meta.Name == decl)).Single(f => f.Meta.Name == name);

	[Fact]
	public void ResolvesThePreludeShapes()
	{
		var s = Catalog();

		var name = s.Resolve(Field(s, "Product", "name").Type);
		Assert.Equal((Shape.Prim, "string"), (name.Shape, name.Name));

		var category = s.Resolve(Field(s, "Product", "category").Type);
		Assert.Equal(Shape.Option, category.Shape);
		Assert.Equal("string", category.Elem?.Name);

		var tags = s.Resolve(Field(s, "Product", "tags").Type);
		Assert.Equal(Shape.Set, tags.Shape);
		Assert.Equal(Shape.Prim, tags.Elem?.Shape);
	}

	[Fact]
	public void RefusesAnUnresolvedType()
	{
		var s = Catalog();

		var e = Assert.Throws<ProblemException>(() => s.Resolve(new ID { Index = -1, Name = "Missing" }));
		Assert.Contains("type Missing did not resolve", e.Problem.Message, StringComparison.Ordinal);
	}

	[Fact]
	public void ReadsAConstraintsSource()
	{
		var s = Catalog();

		var length = Assert.Single(Field(s, "Product", "name").Constraints);
		Assert.Equal("length(1..64)", s.Source(length));
	}

	[Theory]
	[InlineData(LiteralKind.String, "a \"b\"\n", "\"a \\\"b\\\"\\n\"")]
	[InlineData(LiteralKind.Regex, "^a+$", "/^a+$/")]
	[InlineData(LiteralKind.Int, "-2", "-2")]
	[InlineData(LiteralKind.Name, "Pending", "Pending")]
	public void WritesALiteralAsSourceDoes(LiteralKind kind, string text, string want) =>
		Assert.Equal(want, Literals.Text(new Literal { Kind = kind, Text = text }));

	[Fact]
	public void WritesAListAndARange()
	{
		var list = new Literal { Kind = LiteralKind.List, Items = { new Literal { Kind = LiteralKind.String, Text = "a" }, new Literal { Kind = LiteralKind.Int, Text = "1" } } };
		var range = new Literal { Kind = LiteralKind.Range, Range = new Ir.V1.Range { Low = 1 } };

		Assert.Equal("[\"a\", 1]", Literals.Text(list));
		Assert.Equal("1..", Literals.Text(range));
		Assert.Equal("a float", Literals.KindName(LiteralKind.Float));
	}
}
