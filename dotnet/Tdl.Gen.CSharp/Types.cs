using Tdl.DotNet;
using Tdl.Ir.V1;
using Tdl.Plugin;

namespace Tdl.Gen.CSharp;

/// <summary>
/// A rendered type. Cmp is the comparer that compares it by content, or null
/// when EqualityComparer&lt;T&gt;.Default already does.
/// </summary>
sealed record CsType(string Text, string? Cmp = null)
{
	/// <summary>The comparer a record compares a property of type text by.</summary>
	public static string DefaultComparer(string text) => $"global::System.Collections.Generic.EqualityComparer<{text}>.Default";

	/// <summary>The expression comparing a value of the type by content.</summary>
	public string Comparer => Cmp ?? DefaultComparer(Text);
}

sealed partial class Generator
{
	const string Immutable = "global::System.Collections.Immutable.";
	const string ListType = Immutable + "IImmutableList";
	const string SetType = Immutable + "IImmutableSet";
	const string MapType = Immutable + "IImmutableDictionary";

	// KeywordTypes is the C# keyword spelling each base class library type has.
	static readonly Dictionary<string, string> KeywordTypes = new()
	{
		["String"] = "string",
		["Int64"] = "long",
		["Int32"] = "int",
		["UInt32"] = "uint",
		["UInt64"] = "ulong",
		["Single"] = "float",
		["Double"] = "double",
		["Boolean"] = "bool",
		["Byte"] = "byte",
		["Decimal"] = "decimal",
	};

	// FieldType renders a field's type and whether the property is required:
	// `T?` may be left out, `T | null` must be written and may be null.
	(CsType, bool) FieldType(Field f, ISet<string>? shadow)
	{
		var r = s.Resolve(f.Type);
		var required = r.Shape switch
		{
			Shape.Option => false,
			Shape.Nullable => r.Elem!.Shape != Shape.Option,
			_ => true,
		};
		return (Render(r, shadow), required);
	}

	// Type renders a type reference.
	CsType Type(ID id, ISet<string>? shadow) => Render(s.Resolve(id), shadow);

	// Render renders a resolved type. A type whose name a nested variant
	// shadows, named in shadow, is written qualified.
	CsType Render(Ref r, ISet<string>? shadow)
	{
		switch (r.Shape)
		{
			case Shape.Prim:
				if (ForeignPrim(r.Id) is { } mapped)
				{
					return new CsType(mapped);
				}
				if (s.Model.Types_[r.Id.Index].Args.Count > 0)
				{
					s.Lossy(LossCodes.Unit, r.Position, $"C# has no units of measure, so {r.Id.Name} is written as its base type");
				}
				if (!Net.Primitive(r.Name, out var t))
				{
					throw Unsupported(r.Position, $"primitive {r.Name} has no C# type", LossCodes.Primitive);
				}
				var text = Spell(t);
				return t.Array ? new CsType(text + "[]", EqualityClass() + ".Bytes") : new CsType(text);

			case Shape.Option or Shape.Nullable:
				var elem = r.Elem!;
				if (elem.Shape is Shape.Option or Shape.Nullable)
				{
					// C# has one null, so an absent value and a null one are the
					// same.
					s.Lossy(LossCodes.Optional, r.Position, "C# cannot tell an absent value from a null one, so both are null");
					while (elem.Shape is Shape.Option or Shape.Nullable)
					{
						elem = elem.Elem!;
					}
				}
				var inner = Render(elem, shadow);
				return inner with { Text = inner.Text + "?" };

			case Shape.List:
				var item = Render(r.Elem!, shadow);
				return new CsType($"{ListType}<{item.Text}>", $"{EqualityClass()}.List<{item.Text}>({item.Comparer})");

			case Shape.Set:
				Hashed(r.Elem!, "a set's element");
				var member = Render(r.Elem!, shadow);
				return new CsType($"{SetType}<{member.Text}>", $"{EqualityClass()}.Set<{member.Text}>()");

			case Shape.Map:
				Hashed(r.Key!, "a map's key");
				var key = Render(r.Key!, shadow);
				var val = Render(r.Elem!, shadow);
				return new CsType($"{MapType}<{key.Text}, {val.Text}>", $"{EqualityClass()}.Map<{key.Text}, {val.Text}>({val.Comparer})");

			case Shape.Extern:
				if (!externs.TryGetValue(r.Extern!, out var ext))
				{
					throw Unsupported(r.Position,
						$"{r.Extern!.Package}.{r.Extern.Name} has no C# type: give it a foreign directive, or its package a csharp target block");
				}
				return new CsType(ext);

			case Shape.Named:
				if (r.Decl is { } d && foreign.TryGetValue(d, out var f))
				{
					return new CsType(f);
				}
				if (r.Decl is null || !names.TryGetValue(r.Decl, out var name))
				{
					// A declaration that does not claim a name is skipped, and the
					// cascade skips this one.
					throw Unsupported(r.Position, $"{r.Decl?.Meta?.Name} is not generated");
				}
				return new CsType(shadow?.Contains(name) == true ? $"global::{ns}.{name}" : name);
		}
		throw Unsupported(r.Position, $"type {r.Id.Name} has no C# type");
	}

	// Hashed refuses, as a set's element or a map's key, a type .NET hashes
	// by reference or one that may be null.
	static void Hashed(Ref r, string what)
	{
		if (r.Shape is Shape.Option or Shape.Nullable)
		{
			throw Unsupported(r.Position, $"{what} cannot be optional in C#");
		}
		if (r.Shape is Shape.List or Shape.Set or Shape.Map)
		{
			throw Unsupported(r.Position, $"{what} cannot be a collection in C#, which hashes it by reference");
		}
		if (r is { Shape: Shape.Prim, Name: "bytes" })
		{
			throw Unsupported(r.Position, $"{what} cannot be bytes in C#, which hashes it by reference");
		}
	}

	// Spell is a base class library type as C# writes it.
	static string Spell(BclType t) =>
		t.Namespace == "System" && KeywordTypes.TryGetValue(t.Name, out var k) ? k : $"global::{t.Namespace}.{t.Name}";

	// Equality is the namespace's equality class, qualified, since a nested
	// variant may shadow it.
	string EqualityClass() => $"global::{ns}.{EqualityType}";
}
