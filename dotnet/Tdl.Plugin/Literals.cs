using System.Globalization;
using System.Text;
using Tdl.Ir.V1;

namespace Tdl.Plugin;

/// <summary>Reads literals as TDL source writes them.</summary>
public static class Literals
{
	/// <summary>
	/// A literal as TDL source writes it: a string quoted, a regex between
	/// slashes, a list bracketed, and a range with its bounds.
	/// </summary>
	public static string Text(Literal l) => l.Kind switch
	{
		LiteralKind.String => Quote(l.Text),
		LiteralKind.Regex => "/" + l.Text + "/",
		LiteralKind.List => "[" + string.Join(", ", l.Items.Select(Text)) + "]",
		LiteralKind.Range => l.Range is { } r
			? (r.HasLow ? r.Low.ToString(CultureInfo.InvariantCulture) : "") + ".." + (r.HasHigh ? r.High.ToString(CultureInfo.InvariantCulture) : "")
			: "",
		_ => l.Text,
	};

	/// <summary>A literal kind as a message names it: a string, an integer.</summary>
	public static string KindName(LiteralKind k) => k switch
	{
		LiteralKind.String => "a string",
		LiteralKind.Int => "an integer",
		LiteralKind.Float => "a float",
		LiteralKind.Bool => "a boolean",
		LiteralKind.Name => "a name",
		LiteralKind.Regex => "a regex",
		LiteralKind.List => "a list",
		LiteralKind.Range => "a range",
		LiteralKind.Unspecified => "an unspecified value",
		// A kind from a newer schema.
		_ => "an unrecognized value",
	};

	/// <summary>A string quoted with TDL's escapes, as the host quotes one.</summary>
	public static string Quote(string s)
	{
		var b = new StringBuilder("\"");
		foreach (var r in s.EnumerateRunes())
		{
			switch (r.Value)
			{
				case '"': b.Append("\\\""); break;
				case '\\': b.Append("\\\\"); break;
				case '\a': b.Append("\\a"); break;
				case '\b': b.Append("\\b"); break;
				case '\f': b.Append("\\f"); break;
				case '\n': b.Append("\\n"); break;
				case '\r': b.Append("\\r"); break;
				case '\t': b.Append("\\t"); break;
				case '\v': b.Append("\\v"); break;
				case < 0x20 or 0x7f: b.Append(CultureInfo.InvariantCulture, $"\\x{r.Value:x2}"); break;
				default: b.Append(r.ToString()); break;
			}
		}
		return b.Append('"').ToString();
	}
}
