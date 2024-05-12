using System.CommandLine;
using System.Reflection;
using Serilog;

namespace UnMango.Tdl.Cli.Broker;

internal static class Handlers
{
	public static Task Start(IBroker broker, CancellationToken cancellationToken) {
		return broker.Start(cancellationToken).AsTask();
	}

	public static async Task Status(IBroker broker, IConsole console, CancellationToken cancellationToken) {
		var status = await broker.Status(cancellationToken);
		console.WriteLine($"Version: {status.Version}");
		console.WriteLine($"State:   {status.State}");
	}

	public static Task Stop(IBroker broker, CancellationToken cancellationToken) {
		return broker.Stop(cancellationToken).AsTask();
	}

	public static Task Upgrade(IBroker broker, CancellationToken cancellationToken) {
		var entryAssembly = Assembly.GetEntryAssembly();
		if (entryAssembly is null) {
			Log.Error("Failed to retrieve entry assembly");
			return Task.CompletedTask;
		}

		var version = entryAssembly.GetName().Version;
		if (version is null) {
			Log.Error("Failed to retrieve version");
			return Task.CompletedTask;
		}

		Log.Verbose("Upgrading broker");
		return broker.Upgrade(version.ToString(), cancellationToken);
	}
}
