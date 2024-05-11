using UnMango.Tdl.Cli.Docker;
using UnMango.Tdl.Cli.Internal;

namespace UnMango.Tdl.Cli.Broker;

internal static class Handlers
{
	public static Task Start(CancellationToken cancellationToken) {
		Console.WriteLine("Not implemented");
		return Task.CompletedTask;
	}

	public static async Task Status(IDocker docker, CancellationToken cancellationToken) {
		var labels = new Dictionary<string, string> {
			[Config.OwnerLabel] = Constants.Owner,
		};

		var container = await docker.FindMatching(labels, cancellationToken);

		if (container is null) {
			Console.WriteLine("Unable to find matching container: {0}", labels);
			return;
		}

		var inspection = await docker.Inspect(container, cancellationToken);
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
