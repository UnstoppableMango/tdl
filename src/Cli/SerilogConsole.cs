using System.CommandLine;
using System.CommandLine.IO;
using Microsoft.Extensions.Logging;

namespace UnMango.Tdl.Cli;

file sealed record SerilogStreamWriter(ILogger Logger) : IStandardStreamWriter
{
	public void Write(string? value) => Logger.LogInformation("{Value}", value);
}

internal sealed record SerilogConsole(ILogger<Program> Logger, IConsole Console) : IConsole
{
	public IStandardStreamWriter Out { get; } = new SerilogStreamWriter(Logger);
	public IStandardStreamWriter Error { get; } = new SerilogStreamWriter(Logger);

	public bool IsOutputRedirected => Console.IsOutputRedirected;
	public bool IsErrorRedirected => Console.IsErrorRedirected;
	public bool IsInputRedirected => Console.IsInputRedirected;
}
