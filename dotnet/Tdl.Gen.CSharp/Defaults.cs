using System.Globalization;
using System.Numerics;
using Microsoft.CodeAnalysis.CSharp;
using Tdl.Ir.V1;
using Tdl.Plugin;

namespace Tdl.Gen.CSharp;

sealed partial class Generator
{
	// DefaultValue renders a field's default as the property's initializer.
	// A problem is a default C# cannot write, which the caller warns at and
	// drops, leaving the property required.
	string DefaultValue(ID id, Literal lit, ISet<string>? shadow) => Literal(s.Resolve(id), lit, shadow);

	// Literal renders lit as a value of the type r.
	string Literal(Ref r, Literal lit, ISet<string>? shadow)
	{
		var kind = lit.Kind;
		var text = lit.Text;
		ProblemException Drop(string why) =>
			Unsupported(lit.Position, $"the default {Literals.Text(lit)} is dropped: {why}", LossCodes.Default);

		switch (r.Shape)
		{
			case Shape.Option or Shape.Nullable:
				return Literal(r.Elem!, lit, shadow);

			case Shape.Extern:
				throw Drop($"{r.Extern!.Package}.{r.Extern.Name} is a foreign type, whose values another assembly decides");

			case Shape.Prim:
				if (ForeignPrim(r.Id) is not null)
				{
					throw Drop($"{r.Name} is a foreign type, whose values another assembly decides");
				}
				switch (r.Name)
				{
					case "string" when kind == LiteralKind.String:
						return SymbolDisplay.FormatLiteral(text, quote: true);
					case "string":
						break;
					case "bool" when kind == LiteralKind.Bool:
						return text;
					case "bool":
						break;
					case "int" or "int64" or "int32" or "uint32" or "uint64" when kind == LiteralKind.Int:
						return Fits(r.Name, text) ? text : throw Drop($"it is out of range for {r.Name}");
					case "int" or "int64" or "int32" or "uint32" or "uint64":
						break;
					case "float32" or "float64" or "decimal" when kind is LiteralKind.Int or LiteralKind.Float:
						if (!FitsReal(r.Name, text))
						{
							throw Drop($"it is out of range for {r.Name}");
						}
						return r.Name switch
						{
							"float32" => text + "f",
							"decimal" => text + "m",
							_ when kind == LiteralKind.Int => text + ".0",
							_ => text,
						};
					case "float32" or "float64" or "decimal":
						break;
					default:
						throw Drop($"C# has no constant of type {r.Name}");
				}
				break;

			case Shape.List or Shape.Set when kind == LiteralKind.List:
				var elem = Render(r.Elem!, shadow);
				var factory = Immutable + (r.Shape == Shape.Set ? "ImmutableHashSet" : "ImmutableList");
				if (lit.Items.Count == 0)
				{
					return $"{factory}<{elem.Text}>.Empty";
				}
				var items = lit.Items.Select(item => Literal(r.Elem!, item, shadow));
				return $"{factory}.Create<{elem.Text}>({string.Join(", ", items)})";

			case Shape.Map when kind == LiteralKind.List && lit.Items.Count == 0:
				var map = Render(r, shadow);
				return Immutable + "ImmutableDictionary" + map.Text[MapType.Length..] + ".Empty";

			case Shape.Named:
				var d = r.Decl!;
				if (foreign.ContainsKey(d))
				{
					throw Drop($"{d.Meta?.Name} is a foreign type, whose values another assembly decides");
				}
				var t = Render(r, shadow);
				if (d.Newtype is not null)
				{
					// A newtype's base is written from the top level, where no
					// variant shadows a name.
					var inner = Literal(s.Resolve(d.Newtype.Base), lit, null);
					return $"new {t.Text}({inner})";
				}
				if (d.Enumeration is not null && kind == LiteralKind.Name)
				{
					var v = VariantOf(d.Enumeration, lit) ?? throw Drop($"{d.Meta?.Name} has no variant {text}");
					if (!MemberName(VariantName(v), out var name, out var problem))
					{
						throw Unsupported(v.Meta?.Position, $"{d.Meta?.Name}.{v.Meta?.Name} {problem}");
					}
					if (s.View(d)?.Fielded != true)
					{
						return $"{t.Text}.{name}";
					}
					if (v.Fields.Count > 0)
					{
						throw Drop($"variant {v.Meta?.Name} has fields");
					}
					return $"new {t.Text}.{name}()";
				}
				throw Drop($"C# has no constant of type {d.Meta?.Name}");
		}
		throw Drop("it is not a value of its field's type");
	}

	// VariantOf is the variant a name denotes. Lowering resolves a default
	// written as a name, but not a name inside a list.
	static Variant? VariantOf(Ir.V1.Enum e, Literal lit)
	{
		var vs = e.Variants;
		if (lit.Variant is { } id && id.Index >= 0 && id.Index < vs.Count)
		{
			return vs[id.Index];
		}
		return vs.FirstOrDefault(v => v.Meta?.Name == lit.Text);
	}

	// Fits reports whether an integer literal is in range for the primitive.
	static bool Fits(string prim, string text)
	{
		if (prim.StartsWith('u'))
		{
			return ulong.TryParse(text, NumberStyles.None, CultureInfo.InvariantCulture, out var n) && (prim == "uint64" || n <= uint.MaxValue);
		}
		return long.TryParse(text, NumberStyles.AllowLeadingSign, CultureInfo.InvariantCulture, out var m) && (prim != "int32" || m is >= int.MinValue and <= int.MaxValue);
	}

	// MaxDecimal is the largest value of .NET's decimal, 2^96 - 1.
	static readonly BigInteger MaxDecimal = BigInteger.Parse("79228162514264337593543950335", CultureInfo.InvariantCulture);

	// FitsReal reports whether a numeric literal is in range for float32,
	// float64, or decimal, which C# otherwise refuses with CS0594.
	static bool FitsReal(string prim, string text)
	{
		const NumberStyles style = NumberStyles.Float;
		if (prim == "decimal")
		{
			if (!Decimal(text, out var digits, out var scale))
			{
				return false;
			}
			// The magnitude is below 10^(length + scale), and the maximum has
			// 29 digits, so only a literal near it is compared exactly.
			var magnitude = (long)digits.ToString(CultureInfo.InvariantCulture).Length + scale;
			return digits.IsZero || magnitude < 29 || magnitude <= 30 &&
				(scale >= 0 ? digits * BigInteger.Pow(10, scale) <= MaxDecimal : digits <= MaxDecimal * BigInteger.Pow(10, -scale));
		}
		return prim == "float32"
			? float.TryParse(text, style, CultureInfo.InvariantCulture, out var f) && float.IsFinite(f)
			: double.TryParse(text, style, CultureInfo.InvariantCulture, out var d) && double.IsFinite(d);
	}

	// Decimal reads a numeric literal's magnitude as digits times ten to the
	// scale.
	static bool Decimal(string text, out BigInteger digits, out int scale)
	{
		var mantissa = text.TrimStart('-', '+');
		scale = 0;
		if (mantissa.IndexOfAny(['e', 'E']) is var e and >= 0)
		{
			if (!int.TryParse(mantissa[(e + 1)..], NumberStyles.AllowLeadingSign, CultureInfo.InvariantCulture, out scale))
			{
				digits = default;
				return false;
			}
			mantissa = mantissa[..e];
		}
		var parts = mantissa.Split('.');
		if (parts.Length > 2)
		{
			digits = default;
			return false;
		}
		if (parts.Length == 2)
		{
			scale -= parts[1].Length;
		}
		var all = string.Concat(parts);
		return BigInteger.TryParse(all == "" ? "0" : all, NumberStyles.None, CultureInfo.InvariantCulture, out digits);
	}
}
