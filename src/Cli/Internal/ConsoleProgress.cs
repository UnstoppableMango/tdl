using System.CommandLine;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

internal sealed record ConsoleProgress(IConsole Console) : IDockerProgress
{
	public void Report(JSONMessage value) {
		// Console.Write("Got progress: ");
		// Console.WriteLine(value.ProgressMessage);
		// Console.WriteLine(value.ErrorMessage);
		Console.Write(value.Status);
	}

	public void Report(string value) => Console.WriteLine(value);
}