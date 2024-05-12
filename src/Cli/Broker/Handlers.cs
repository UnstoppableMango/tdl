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

	public static Task Upgrade(IBroker broker, bool latest, string? version, CancellationToken cancellationToken) {
		if (latest) {
			Log.Verbose("Using latest version");
			return broker.Upgrade(cancellationToken);
		}

		if (string.IsNullOrWhiteSpace(version)) {
			Log.Verbose("Using assembly version");
			version = GetAssemblyVersion();
		}

		Log.Verbose("Upgrading broker");
		return broker.Upgrade(version, cancellationToken);
	}

	private static string? GetAssemblyVersion() {
		var assembly = Assembly.GetEntryAssembly();
		if (assembly is null) {
			Log.Debug("Failed to retrieve entry assembly");
			return null;
		}

		var version = assembly.GetName().Version;
		if (version is not null)
			return version.ToString();

		Log.Debug("Failed to retrieve assembly version");
		return null;
	}
}
