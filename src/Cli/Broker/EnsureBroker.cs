using System.CommandLine.Invocation;
using System.CommandLine.Parsing;
using Microsoft.Extensions.DependencyInjection;

namespace UnMango.Tdl.Cli.Broker;

internal static class EnsureBroker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		var parentCommand = context.ParseResult.CommandResult.Parent;
		if (parentCommand is not CommandResult { Command.Name: "broker" }) {
			var broker = context.BindingContext.GetRequiredService<IBroker>();
			await broker.Start(context.GetCancellationToken());
		}

		await next(context);
	};
}
