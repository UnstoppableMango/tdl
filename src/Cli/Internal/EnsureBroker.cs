using System.CommandLine.Invocation;
using Serilog;

namespace UnMango.Tdl.Cli.Internal;

using Microsoft.Extensions.DependencyInjection;

internal static class EnsureBroker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		var cancellationToken = context.GetCancellationToken();
		var docker = context.BindingContext.GetRequiredService<IDocker>();

		// TODO: Check local env and build if needed
		var start = await docker.Start(new StartArgs {
			Image = $"{Config.ContainerRepo}/tdl-broker",
			Tag = Config.ContainerTag,
			Name = "tdl-test",
			Volumes = [$"{Config.SocketDir}:/var/run/tdl"],
		}, cancellationToken);

		try {
			await next(context);
		}
		finally {
			await docker.Stop(start, cancellationToken);
		}
	};
}
