using System.CommandLine.Invocation;
using Microsoft.Extensions.DependencyInjection;
using Serilog;

namespace UnMango.Tdl.Cli.Broker;

internal static class EnsureBroker
{
	public static InvocationMiddleware Middleware => async (context, next) => {
		if (context.ParseResult.CommandResult.Command.Name == "broker") {
			Log.Verbose("Skipping broker start");
			return;
		}

		var broker = context.BindingContext.GetRequiredService<IBroker>();
		await broker.Start(context.GetCancellationToken());
		await next(context);
	};
}
