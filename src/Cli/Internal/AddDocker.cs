using System.CommandLine.Invocation;
using Docker.DotNet;

namespace UnMango.Tdl.Cli.Internal;

internal static class AddDocker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		var progress = new ConsoleProgress(context.Console);
		using var client = new DockerClientConfiguration().CreateClient();
		var docker = new Docker(client, progress);
		context.BindingContext.AddService<IDocker>(_ => docker);
		await next(context);
	};
}
