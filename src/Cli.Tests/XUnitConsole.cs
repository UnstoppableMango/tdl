using System.CommandLine;
using System.CommandLine.IO;
using Xunit.Abstractions;

namespace UnMango.Tdl.Cli.Tests;

public sealed class XUnitConsole(ITestOutputHelper test) : IConsole
{
	public IStandardStreamWriter Out { get; } = new StreamWriter(test);
	public IStandardStreamWriter Error { get; } = new StreamWriter(test);

	public bool IsOutputRedirected => true;
	public bool IsErrorRedirected => true;
	public bool IsInputRedirected => true;

	private class StreamWriter(ITestOutputHelper test) : IStandardStreamWriter
	{
		public void Write(string? value) => test.WriteLine(value);
	}
}
