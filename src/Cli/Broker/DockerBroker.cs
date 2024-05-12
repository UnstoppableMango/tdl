using System.Text.RegularExpressions;
using Serilog;
using UnMango.Tdl.Cli.Docker;

namespace UnMango.Tdl.Cli.Broker;

internal static partial class Patterns
{
	[GeneratedRegex(".*Application started.*")]
	public static partial Regex ApplicationStarted();
}

internal sealed class DockerBroker(IDocker docker) : IBroker
{
	private const string OwnerLabel = "tdl.owner", Owner = "tdl-cli";
	private static readonly Regex ApplicationStarted = Patterns.ApplicationStarted();

	private static readonly IDictionary<string, string> Labels = new Dictionary<string, string> {
		[Config.OwnerLabel] = Owner,
	};

	public async ValueTask<bool> Running(CancellationToken cancellationToken) {
		if (File.Exists(Config.Socket)) {
			return true;
		}

		var container = await docker.FindMatching(Labels, cancellationToken);
		return container != null;
	}

	public async ValueTask Start(CancellationToken cancellationToken) {
		if (await Running(cancellationToken)) {
			Log.Verbose("Broker is already running");
			return;
		}

		var uid = await Config.Uid();
		var gid = await Config.Gid();

		Log.Debug("Starting broker");
		var container = await docker.Start(new StartArgs {
			Image = $"{Config.ContainerRepo}/tdl-broker",
			Tag = Config.ContainerTag,
			User = $"{uid}:{gid}",
			Volumes = [$"{Config.SocketDir}:/var/run/tdl"],
			Tmpfs = ["/app/plugins"],
			Labels = { [OwnerLabel] = Owner },
		}, cancellationToken);
		Log.Verbose("Started broker");

		_ = docker.FollowLogs(container, cancellationToken);
		await docker.WaitFor(container, ApplicationStarted.IsMatch, cancellationToken);
	}

	public async Task<BrokerStatus> Status(CancellationToken cancellationToken) {
		var container = await docker.FindMatching(Labels, cancellationToken);
		if (container is null) {
			return new BrokerStatus {
				State = "not found",
				Version = "unknown",
			};
		}

		var inspection = await docker.Inspect(container, cancellationToken);
		return new BrokerStatus {
			Version = inspection.Version,
			State = inspection.State,
		};
	}

	public async ValueTask Stop(CancellationToken cancellationToken) {
		if (!await Running(cancellationToken))
			return;

		var container = await docker.FindMatching(Labels, cancellationToken);
		if (container is null)
			return;

		await docker.Stop(container, cancellationToken);
	}

	public Task Upgrade(string? version, CancellationToken cancellationToken) {
		throw new NotImplementedException();
	}
}
