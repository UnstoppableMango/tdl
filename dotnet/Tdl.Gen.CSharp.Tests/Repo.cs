using Tdl.Plugin.V1;

namespace Tdl.Gen.CSharp.Tests;

// Repo finds the requests tdl recorded for the C# backend, which
// internal/gen/csharp_test.go writes, from the test assembly's directory.
static class Repo
{
	public static string Root { get; } = FindRoot();

	static string FindRoot()
	{
		for (var dir = new DirectoryInfo(AppContext.BaseDirectory); dir is not null; dir = dir.Parent)
		{
			if (Directory.Exists(Path.Combine(dir.FullName, "testdata", "csharp")))
			{
				return dir.FullName;
			}
		}
		throw new DirectoryNotFoundException("no testdata/csharp above " + AppContext.BaseDirectory);
	}

	public static string Corpus { get; } = Path.Combine(Root, "testdata", "csharp", "corpus");

	// Case is the request recorded for testdata/csharp/cases/<name>.tdl.
	public static Request Case(string name) =>
		Request.Parser.ParseFrom(System.IO.File.ReadAllBytes(Path.Combine(Root, "testdata", "csharp", "cases", name + ".request.binpb")));

	// CorpusCases are the directories of testdata/csharp/corpus, for a theory.
	public static TheoryData<string> CorpusCases()
	{
		var data = new TheoryData<string>();
		foreach (var dir in Directory.GetDirectories(Corpus).Order())
		{
			data.Add(Path.GetFileName(dir));
		}
		return data;
	}
}
