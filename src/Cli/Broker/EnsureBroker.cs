using System.CommandLine.Invocation;
using System.Text.RegularExpressions;
using Microsoft.Extensions.DependencyInjection;
using Serilog;
using UnMango.Tdl.Cli.Docker;

namespace UnMango.Tdl.Cli.Broker;

internal static partial class Patterns
{
	[GeneratedRegex(".*Application started.*")]
	public static partial Regex ApplicationStarted();
}

internal static class EnsureBroker
{
	private const string OwnerLabel = "tdl.owner", Owner = "tdl-cli";
	private static readonly Regex ApplicationStarted = Patterns.ApplicationStarted();

	public static InvocationMiddleware Middleware => async (context, next) => {
		var docker = context.BindingContext.GetRequiredService<IDocker>();
		await EnsureStarted(docker, context.GetCancellationToken());
		await next(context);
	};

	private static Task EnsureStarted(IDocker docker, CancellationToken cancellationToken) {
		Log.Verbose("Checking for socket existence");
		if (!File.Exists(Config.Socket))
			return Start(docker, cancellationToken);

		Log.Debug("Socket exists");
		return Task.CompletedTask;
	}

	private static async Task Start(IDocker docker, CancellationToken cancellationToken) {
		var uid = await Config.Uid();
		var gid = await Config.Gid();

		Log.Debug("Starting broker");
		var container = await docker.Start(new StartArgs {
			Image = $"{Config.ContainerRepo}/tdl-broker",
			Tag = Config.ContainerTag,
			User = $"{uid}:{gid}",
			Volumes = [$"{Config.SocketDir}:/var/run/tdl"],
			Labels = { [OwnerLabel] = Owner },
		}, cancellationToken);
		Log.Verbose("Started broker");

		_ = docker.FollowLogs(container, cancellationToken);
		await docker.WaitFor(container, ApplicationStarted.IsMatch, cancellationToken);
	}
}
