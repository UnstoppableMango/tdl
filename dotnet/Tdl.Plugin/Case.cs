using System.Text;

namespace Tdl.Plugin;

/// <summary>
/// A case style joins the words of a name. The analysis ships every
/// declaration's, field's, and variant's words, so every SDK cases a name
/// identically.
/// </summary>
public delegate string Style(IReadOnlyList<string> words);

/// <summary>The case styles backends share.</summary>
public static class Case
{
	/// <summary>
	/// Joins words with each one's first letter upper case and the rest as
	/// written: user and id are UserId, and user and ID are UserID.
	/// </summary>
	public static string Pascal(IReadOnlyList<string> words)
	{
		var b = new StringBuilder();
		foreach (var w in words)
		{
			b.Append(UpperFirst(w));
		}
		return b.ToString();
	}

	/// <summary>Is <see cref="Pascal"/> with the first word entirely lower case.</summary>
	public static string Camel(IReadOnlyList<string> words)
	{
		if (words.Count == 0)
		{
			return "";
		}
		var b = new StringBuilder(words[0].ToLowerInvariant());
		foreach (var w in words.Skip(1))
		{
			b.Append(UpperFirst(w));
		}
		return b.ToString();
	}

	/// <summary>Joins lower-case words with underscores.</summary>
	public static string Snake(IReadOnlyList<string> words) =>
		string.Join("_", words.Select(w => w.ToLowerInvariant()));

	/// <summary>Joins upper-case words with underscores.</summary>
	public static string ScreamingSnake(IReadOnlyList<string> words) =>
		string.Join("_", words.Select(w => w.ToUpperInvariant()));

	/// <summary>
	/// Splits a name into the words a case style joins, as tdl splits the
	/// names it ships: userID is user and ID, HTTPServer is HTTP and Server,
	/// and last4 is one word. Anything that is not a letter or a digit
	/// separates words and is dropped.
	/// </summary>
	public static IReadOnlyList<string> Words(string name)
	{
		var words = new List<string>();
		var cur = new StringBuilder();
		void Flush()
		{
			if (cur.Length > 0)
			{
				words.Add(cur.ToString());
				cur.Clear();
			}
		}

		var runes = name.EnumerateRunes().ToArray();
		var prev = default(Rune);
		for (var i = 0; i < runes.Length; i++)
		{
			var c = runes[i];
			if (!Rune.IsLetter(c) && !Rune.IsDigit(c))
			{
				Flush();
				continue;
			}
			if (Rune.IsUpper(c) && cur.Length > 0)
			{
				var nextLower = i + 1 < runes.Length && Rune.IsLower(runes[i + 1]);
				if (Rune.IsLower(prev) || Rune.IsDigit(prev) || (Rune.IsUpper(prev) && nextLower))
				{
					Flush();
				}
			}
			cur.Append(c.ToString());
			prev = c;
		}
		Flush();
		return words;
	}

	/// <summary>The part of a dotted name after its last dot.</summary>
	public static string LastSegment(string name) =>
		name.LastIndexOf('.') is var i and >= 0 ? name[(i + 1)..] : name;

	/// <summary>Turns SCREAMING_SNAKE into Pascal case: IN_TRANSIT is InTransit.</summary>
	public static string FromScreaming(string s)
	{
		var b = new StringBuilder();
		foreach (var w in s.Split('_'))
		{
			if (w.Length > 0)
			{
				b.Append(UpperFirst(w.ToLowerInvariant()));
			}
		}
		return b.ToString();
	}

	static string UpperFirst(string w)
	{
		if (w.Length == 0)
		{
			return w;
		}
		var first = Rune.GetRuneAt(w, 0);
		return Rune.ToUpperInvariant(first).ToString() + w[first.Utf16SequenceLength..];
	}
}
