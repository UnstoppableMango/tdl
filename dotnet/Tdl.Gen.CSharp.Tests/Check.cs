using System.Reflection;
using System.Runtime.Loader;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Tdl.Plugin.V1;
using Diagnostic = Tdl.Plugin.V1.Diagnostic;

namespace Tdl.Gen.CSharp.Tests;

// Check generates from recorded requests and holds the output to what the
// design promises: the text it holds, the warnings it raises, and that it
// compiles with every warning an error.
static class Check
{
	public static Response Generate(string name) => new CSharpBackend().Generate(Repo.Case(name));

	// File is the response's file at path.
	public static string File(Response resp, string path)
	{
		var f = resp.Files.FirstOrDefault(f => f.Path == path);
		Assert.True(f is not null, $"no file {path} among [{string.Join(", ", resp.Files.Select(f => f.Path))}]; diagnostics = {resp.Diagnostics}");
		return f.Content.ToStringUtf8();
	}

	public static IEnumerable<string> Paths(Response resp) => resp.Files.Select(f => f.Path);

	public static void Contains(string src, params string[] wants)
	{
		foreach (var want in wants)
		{
			Assert.Contains(want, src, StringComparison.Ordinal);
		}
	}

	public static void Absent(string src, params string[] unwanted)
	{
		foreach (var u in unwanted)
		{
			Assert.DoesNotContain(u, src, StringComparison.Ordinal);
		}
	}

	// Warned asserts a warning's message contains each substring.
	public static void Warned(Response resp, params string[] subs) =>
		Assert.True(resp.Diagnostics.Any(d => d.Severity == Severity.Warning && subs.All(s => d.Message.Contains(s, StringComparison.Ordinal))),
			$"no warning containing [{string.Join(", ", subs)}]: {resp.Diagnostics}");

	public static void Quiet(Response resp) => Assert.Empty(resp.Diagnostics);

	public static IEnumerable<Diagnostic> Errors(Response resp) => resp.Diagnostics.Where(d => d.Severity == Severity.Error);

	// The generated files promise to build with C# 12 on .NET 8, with
	// nullable references and documentation checked.
	static readonly CSharpParseOptions Parse = new(LanguageVersion.CSharp12, DocumentationMode.Diagnose);

	static readonly MetadataReference[] References =
		((string)AppContext.GetData("TRUSTED_PLATFORM_ASSEMBLIES")!)
			.Split(Path.PathSeparator)
			.Select(path => (MetadataReference)MetadataReference.CreateFromFile(path))
			.ToArray();

	// Compile compiles every response, with extra sources beside them, and
	// fails on any error or warning. A malformed doc comment fails; an
	// undocumented member does not.
	public static byte[] Compile(IEnumerable<Response> resps, IEnumerable<string>? extra = null)
	{
		var trees = resps
			.SelectMany((r, i) => r.Files.Select(f => CSharpSyntaxTree.ParseText(f.Content.ToStringUtf8(), Parse, path: $"{i}/{f.Path}")))
			.Concat((extra ?? []).Select((src, i) => CSharpSyntaxTree.ParseText(src, Parse, path: $"extra{i}.cs")))
			.ToList();
		var options = new CSharpCompilationOptions(OutputKind.DynamicallyLinkedLibrary)
			.WithNullableContextOptions(NullableContextOptions.Enable)
			.WithGeneralDiagnosticOption(ReportDiagnostic.Error)
			.WithSpecificDiagnosticOptions([new("CS1591", ReportDiagnostic.Suppress)]);
		var compilation = CSharpCompilation.Create("check", trees, References, options);

		using var output = new MemoryStream();
		var result = compilation.Emit(output);
		var problems = result.Diagnostics.Where(d => d.Severity >= DiagnosticSeverity.Warning).ToList();
		Assert.True(result.Success && problems.Count == 0,
			"the generated C# does not compile:\n" + string.Join("\n", problems) + "\n" + Dump(trees));
		return output.ToArray();
	}

	public static void Compile(params Response[] resps) => Compile(resps, null);

	// Run compiles the response with a program declaring a static class
	// Program whose Run method returns what it found, and calls it.
	public static string Run(Response resp, string program)
	{
		var image = Compile([resp], [program]);
		var context = new AssemblyLoadContext("check", isCollectible: true);
		try
		{
			var assembly = context.LoadFromStream(new MemoryStream(image));
			var run = assembly.GetType("Program")!.GetMethod("Run", BindingFlags.Public | BindingFlags.Static)!;
			return (string)run.Invoke(null, null)!;
		}
		finally
		{
			context.Unload();
		}
	}

	static string Dump(IEnumerable<SyntaxTree> trees) =>
		string.Concat(trees.Select(t => $"==> {t.FilePath} <==\n{t}"));
}
