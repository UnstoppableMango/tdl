using System.CommandLine.Invocation;
using Docker.DotNet;

namespace UnMango.Tdl.Cli.Internal;

using Microsoft.Extensions.DependencyInjection;

internal static class EnsureBroker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		var ct = context.GetCancellationToken();
		var docker = context.BindingContext.GetRequiredService<IDocker>();

		// TODO: Check local env and build if needed
		var start = await docker.Start("image", "tag", ["command"], "name", ct);

		try {
			// TODO: How do we make the container IP available
			await next(context);
		}
		finally {
			await docker.Stop(start, ct);
		}
	};
}