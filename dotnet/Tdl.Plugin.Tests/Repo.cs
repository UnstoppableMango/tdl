using Google.Protobuf;
using Tdl.Plugin.V1;

namespace Tdl.Plugin.Tests;

// Repo finds the repository's testdata and docs from the test assembly's
// directory, wherever the build put it.
static class Repo
{
	public static string Root { get; } = FindRoot();

	static string FindRoot()
	{
		for (var dir = new DirectoryInfo(AppContext.BaseDirectory); dir is not null; dir = dir.Parent)
		{
			if (Directory.Exists(Path.Combine(dir.FullName, "testdata", "plugin")))
			{
				return dir.FullName;
			}
		}
		throw new DirectoryNotFoundException("no testdata/plugin above " + AppContext.BaseDirectory);
	}

	public static string PluginData(string name) => Path.Combine(Root, "testdata", "plugin", name);

	// Exchanges are the base names of the recorded exchanges, for a theory.
	public static TheoryData<string> Exchanges()
	{
		var data = new TheoryData<string>();
		foreach (var path in Directory.GetFiles(Path.Combine(Root, "testdata", "plugin"), "*.request.binpb").Order())
		{
			data.Add(Path.GetFileName(path)[..^".request.binpb".Length]);
		}
		return data;
	}

	public static Request Request(string exchange) =>
		V1.Request.Parser.ParseFrom(System.IO.File.ReadAllBytes(PluginData(exchange + ".request.binpb")));

	public static byte[] Response(string exchange) =>
		System.IO.File.ReadAllBytes(PluginData(exchange + ".response.binpb"));

	// Framed is messages as one end of a connection writes them.
	public static byte[] Framed(params IMessage[] messages)
	{
		var stream = new MemoryStream();
		var conn = new Conn(Stream.Null, stream);
		foreach (var m in messages)
		{
			conn.Send(m);
		}
		return stream.ToArray();
	}
}
