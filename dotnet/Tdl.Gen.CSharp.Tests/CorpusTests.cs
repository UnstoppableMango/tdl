using Tdl.Plugin.V1;
using static Tdl.Gen.CSharp.Tests.Check;

namespace Tdl.Gen.CSharp.Tests;

// The conformance corpus and the smoke model, as tdl requested them, each
// generate the files beside their request and compile. go test
// ./internal/gen -record rewrites both, through tdl-gen-csharp on PATH.
public class CorpusTests
{
	public static TheoryData<string> Cases => Repo.CorpusCases();

	[Fact]
	public void ThereAreCases() => Assert.Contains("smoke", Cases.Select(c => c.Data));

	[Theory]
	[MemberData(nameof(Cases))]
	public void GeneratesTheRecordedFiles(string name)
	{
		var dir = Path.Combine(Repo.Corpus, name);
		var req = Request.Parser.ParseFrom(System.IO.File.ReadAllBytes(Path.Combine(dir, "request.binpb")));
		var resp = new CSharpBackend().Generate(req);
		Assert.Empty(Errors(resp));

		var output = Path.Combine(dir, "out");
		string[] want = Directory.Exists(output)
			? [.. Directory.GetFiles(output, "*", SearchOption.AllDirectories).Select(f => Path.GetRelativePath(output, f).Replace('\\', '/')).Order()]
			: [];
		Assert.Equal(want, Paths(resp).Order());
		foreach (var f in resp.Files)
		{
			Assert.Equal(System.IO.File.ReadAllText(Path.Combine(output, f.Path)), f.Content.ToStringUtf8());
		}
		if (resp.Files.Count > 0)
		{
			Compile(resp);
		}
	}
}
