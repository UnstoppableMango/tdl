using System.CommandLine.Invocation;
using Docker.DotNet;

namespace UnMango.Tdl.Cli.Docker;

internal static class AddDocker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		// var progress = new ConsoleProgress(context.Console);
		var progress = new SerilogProgress();
		// Either need to await this method or dispose of this differently
		using var client = new DockerClientConfiguration().CreateClient();
		var docker = new Docker(client, progress);
		context.BindingContext.AddService<IDocker>(_ => docker);
		await next(context);
	};
}
