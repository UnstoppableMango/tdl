using System.CommandLine.Invocation;
using System.Text.RegularExpressions;
using Serilog;

namespace UnMango.Tdl.Cli.Internal;

using Microsoft.Extensions.DependencyInjection;

internal static partial class Patterns
{
	[GeneratedRegex(".*Application started.*")]
	public static partial Regex ApplicationStarted();
}

internal static class EnsureBroker
{
	private static readonly Regex ApplicationStarted = Patterns.ApplicationStarted();

	public static InvocationMiddleware Middleware => async (context, next) => {
		var cancellationToken = context.GetCancellationToken();
		var docker = context.BindingContext.GetRequiredService<IDocker>();
		var uid = await Config.Uid();
		var gid = await Config.Gid();

		Log.Debug("Starting broker");
		var container = await docker.Start(new StartArgs {
			Image = $"{Config.ContainerRepo}/tdl-broker",
			Tag = Config.ContainerTag,
			Name = "tdl-test",
			User = $"{uid}:{gid}",
			Volumes = [$"{Config.SocketDir}:/var/run/tdl"],
		}, cancellationToken);
		Log.Verbose("Started broker");

		try {
			_ = docker.FollowLogs(container);
			await docker.WaitFor(container, ApplicationStarted.IsMatch, cancellationToken);
			Log.Verbose("Invoking next");
			await next(context);
			Log.Verbose("After invoking next");
		}
		finally {
			Log.Debug("Stopping broker");
			await docker.Stop(container, cancellationToken);
		}
	};
}
