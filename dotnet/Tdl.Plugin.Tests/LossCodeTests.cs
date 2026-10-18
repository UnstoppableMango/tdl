using System.Text.RegularExpressions;

namespace Tdl.Plugin.Tests;

public partial class LossCodeTests
{
	[GeneratedRegex(@"^\| `(lossy\.[a-z-]+)` \|", RegexOptions.Multiline)]
	private static partial Regex Row();

	// The loss codes are the ones docs/design/reverse.md tabulates, in its
	// order, as emit.LossCodes is in Go.
	[Fact]
	public void MatchTheDesign()
	{
		var doc = File.ReadAllText(Path.Combine(Repo.Root, "docs", "design", "reverse.md"));

		var documented = Row().Matches(doc).Select(m => m.Groups[1].Value);

		Assert.Equal(documented, LossCodes.All);
	}
}
