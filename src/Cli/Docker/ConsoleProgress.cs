using System.CommandLine;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

// TODO: Batch similar messages like "Downloading", "Extracting", etc.
internal sealed record ConsoleProgress(IConsole Console) : IDockerProgress
{
	public void Report(JSONMessage value) {
		// Console.Write("Got progress: ");
		// Console.WriteLine(value.ProgressMessage);
		// Console.WriteLine(value.ErrorMessage);
		Console.WriteLine(value.Status);
	}

	public void Report(string value) => Console.WriteLine(value);
}
