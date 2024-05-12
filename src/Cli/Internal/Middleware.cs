using System.CommandLine.Invocation;
using Docker.DotNet;
using Microsoft.Extensions.DependencyInjection;
using UnMango.Tdl.Cli.Broker;
using UnMango.Tdl.Cli.Docker;

namespace UnMango.Tdl.Cli.Internal;

internal static class Middleware
{
	public static InvocationMiddleware Services => async (context, next) => {
		// var progress = new ConsoleProgress(context.Console);
		var progress = new SerilogProgress();

		// Either need to await this method or dispose of this differently
		using var client = new DockerClientConfiguration().CreateClient();
		var docker = new Docker.Docker(client, progress);
		context.BindingContext.AddService<IDocker>(_ => docker);
		context.BindingContext.AddService<IBroker>(_ => new DockerBroker(docker, context.Console));

		await next(context);
	};
}
