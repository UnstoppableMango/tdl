using System.Diagnostics.CodeAnalysis;
using System.Globalization;

namespace Tdl.DotNet;

/// <summary>
/// A base class library type, as System and Int64. Array says the primitive
/// is an array of it, as bytes is of System.Byte.
/// </summary>
public sealed record BclType(string Namespace, string Name, bool Array = false);

/// <summary>
/// A type or an attribute another assembly declares, as <c>foreign</c> and
/// <c>attribute</c> name it.
/// </summary>
public sealed record Qualified(string Namespace, string Name);

/// <summary>
/// What the C# and F# backends decide alike because both target .NET. It
/// holds decisions, not syntax; each backend spells a type its own way. See
/// docs/design/csharp-backend.md.
/// </summary>
public static class Net
{
	static readonly Dictionary<string, BclType> Primitives = new()
	{
		["string"] = new("System", "String"),
		["int"] = new("System", "Int64"),
		["int32"] = new("System", "Int32"),
		["uint32"] = new("System", "UInt32"),
		["int64"] = new("System", "Int64"),
		["uint64"] = new("System", "UInt64"),
		["float32"] = new("System", "Single"),
		["float64"] = new("System", "Double"),
		["bool"] = new("System", "Boolean"),
		["bytes"] = new("System", "Byte", Array: true),
		["decimal"] = new("System", "Decimal"),
		["uuid"] = new("System", "Guid"),
		["instant"] = new("System", "DateTimeOffset"),
		["date"] = new("System", "DateOnly"),
		["duration"] = new("System", "TimeSpan"),
	};

	/// <summary>The type a TDL primitive maps to, or false for one .NET has no type for.</summary>
	public static bool Primitive(string name, [NotNullWhen(true)] out BclType? type) =>
		Primitives.TryGetValue(name, out type);

	/// <summary>
	/// The namespace a package derives: its segments, each with its first
	/// letter upper-cased, so shop.billing is Shop.Billing.
	/// </summary>
	public static string Namespace(string package) =>
		string.Join('.', package.Split('.').Select(UpperFirst));

	/// <summary>A name with its first letter upper-cased.</summary>
	public static string UpperFirst(string s) =>
		s.Length == 0 ? s : char.ToUpper(s[0], CultureInfo.InvariantCulture) + s[1..];

	/// <summary>A name with its first letter lower-cased.</summary>
	public static string LowerFirst(string s) =>
		s.Length == 0 ? s : char.ToLower(s[0], CultureInfo.InvariantCulture) + s[1..];

	/// <summary>
	/// Reads the namespace and name a <c>foreign</c> or <c>attribute</c>
	/// directive gives, or says what is wrong with them. The namespace is
	/// dotted identifiers and the name one identifier; each backend escapes
	/// a segment its language reserves.
	/// </summary>
	public static bool Parse(IReadOnlyList<string> args, [NotNullWhen(true)] out Qualified? q, [NotNullWhen(false)] out string? problem)
	{
		q = null;
		if (args.Count != 2)
		{
			problem = $"takes a namespace and a name, and {args.Count} argument(s) were given";
			return false;
		}
		if (!args[0].Split('.').All(Identifier))
		{
			problem = $"names the namespace \"{args[0]}\", which is not a .NET namespace";
			return false;
		}
		if (!Identifier(args[1]))
		{
			problem = $"names \"{args[1]}\", which is not a .NET identifier";
			return false;
		}
		q = new(args[0], args[1]);
		problem = null;
		return true;
	}

	/// <summary>
	/// Whether s is an identifier both C# and F# accept, before either
	/// escapes a keyword: a letter or underscore, then letters, digits, and
	/// underscores.
	/// </summary>
	public static bool Identifier(string s) =>
		s.Length > 0 &&
		(s[0] == '_' || char.IsLetter(s[0])) &&
		s.All(c => c == '_' || char.IsLetterOrDigit(c));
}
