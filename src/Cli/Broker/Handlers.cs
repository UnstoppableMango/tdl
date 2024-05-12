using System.CommandLine;
using UnMango.Tdl.Cli.Docker;
using UnMango.Tdl.Cli.Internal;

namespace UnMango.Tdl.Cli.Broker;

internal static class Handlers
{
	public static Task Start(CancellationToken cancellationToken) {
		Console.WriteLine("Not implemented");
		return Task.CompletedTask;
	}

	public static async Task Status(IDocker docker, IConsole console, CancellationToken cancellationToken) {
		var labels = new Dictionary<string, string> {
			[Config.OwnerLabel] = Constants.Owner,
		};

		var container = await docker.FindMatching(labels, cancellationToken);
		if (container is null) {
			console.WriteLine($"Unable to find matching container: {labels}");
			return;
		}

		var inspection = await docker.Inspect(container, cancellationToken);
		console.WriteLine($"Version : {inspection.Version}");
		console.WriteLine($"State   : {inspection.State}");
	}

	public static Task Stop(CancellationToken cancellationToken) {
		Console.WriteLine("Not implemented");
		return Task.CompletedTask;
	}

	public static Task Upgrade(CancellationToken cancellationToken) {
		Console.WriteLine("Not implemented");
		return Task.CompletedTask;
	}
}
