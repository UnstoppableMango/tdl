using System.Diagnostics.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Tdl.DotNet;
using Tdl.Ir.V1;
using Tdl.Plugin;

namespace Tdl.Gen.CSharp;

sealed partial class Generator
{
	// ReservedMembers are the members every record declares, which no
	// property or variant may be named.
	static readonly HashSet<string> ReservedMembers =
	[
		"Equals", "GetHashCode", "ToString", "GetType", "Deconstruct", "EqualityContract",
		"PrintMembers", "MemberwiseClone", "Finalize",
	];

	// Keyword reports whether C# reserves s, so an identifier spells it with `@`.
	static bool Keyword(string s) => SyntaxFacts.GetKeywordKind(s) != SyntaxKind.None;

	// Escape writes a keyword with `@`.
	static string Escape(string s) => Keyword(s) ? "@" + s : s;

	// Bare is a name without the `@` escaping a keyword.
	static string Bare(string s) => s.StartsWith('@') ? s[1..] : s;

	// Namespace is the target block's `namespace` directive, or else the
	// package's segments with each first letter upper-cased. It is an error
	// when there is neither or C# refuses the name.
	string Namespace()
	{
		var name = "";
		Position? pos = null;
		if (s.Block("namespace") is { } dir)
		{
			(name, pos) = (dir.Args[0].Text, dir.Position);
		}
		else if (s.Model.Package != "")
		{
			name = Net.Namespace(s.Model.Package);
		}
		if (name == "")
		{
			throw new ProblemException(new Invalid("a C# namespace needs a name: give the model a package, or the csharp target block a namespace directive", pos));
		}
		if (name.Split('.').Any(seg => !SyntaxFacts.IsValidIdentifier(seg) || Keyword(seg) || seg == "global"))
		{
			throw new ProblemException(new Invalid($"{name} is not a C# namespace", pos));
		}
		return name;
	}

	// TypeName is a declaration's C# type name.
	string TypeName(Decl d) =>
		MemberName(s.Text(d.Directives, "name") ?? Net.UpperFirst(Case.LastSegment(d.Meta?.Name ?? "")), out var n, out var problem)
			? n
			: throw Unsupported(d.Meta?.Position, $"{d.Meta?.Name} {problem}");

	// A field's and a variant's names keep their spelling but for the first
	// letter, which is upper-cased.
	string FieldName(Field f) => s.Text(f.Directives, "name") ?? Net.UpperFirst(f.Meta?.Name ?? "");

	string VariantName(Variant v) => s.Text(v.Directives, "name") ?? Net.UpperFirst(v.Meta?.Name ?? "");

	// MemberName checks a name C# would declare, and escapes a keyword.
	static bool MemberName(string n, [NotNullWhen(true)] out string? name, [NotNullWhen(false)] out string? problem)
	{
		if (!SyntaxFacts.IsValidIdentifier(n))
		{
			(name, problem) = (null, $"would be named {n} in C#, which is not an identifier");
			return false;
		}
		(name, problem) = (Escape(n), null);
		return true;
	}

	// VariantName is a variant's C# name, refusing one its enum's scope
	// cannot hold. seen holds the names taken before it.
	string VariantName(Decl d, string typ, Variant v, ISet<string> seen)
	{
		var pos = v.Meta?.Position;
		var owner = $"{d.Meta?.Name}.{v.Meta?.Name}";
		if (!MemberName(VariantName(v), out var n, out var problem))
		{
			throw Unsupported(pos, $"{owner} {problem}");
		}
		var bare = Bare(n);
		if (bare == Bare(typ))
		{
			throw Unsupported(pos, $"{owner} would be named {n} in C#, which a member of {typ} cannot be named; a name directive resolves it");
		}
		if (ReservedMembers.Contains(bare))
		{
			throw Unsupported(pos, $"{owner} would be named {n} in C#, which every type declares; a name directive resolves it");
		}
		if (!seen.Add(bare))
		{
			throw Unsupported(pos, $"{owner} would be named {n} in C#, and another variant already is; a name directive resolves it");
		}
		return n;
	}
}
