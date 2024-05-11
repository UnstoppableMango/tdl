using System.CommandLine;
using System.CommandLine.Builder;
using System.CommandLine.Invocation;
using System.CommandLine.Parsing;
using Serilog;
using Serilog.Events;
using UnMango.Tdl;
using UnMango.Tdl.Cli;
using UnMango.Tdl.Cli.Internal;

var root = new RootCommand("UnstoppableMango's Type Description Language CLI") {
	Commands.Gen(),
	Commands.From(),
};

Log.Logger = new LoggerConfiguration()
	.Enrich.FromLogContext()
	.WriteTo.Console(restrictedToMinimumLevel: ConsoleLogLevel())
	.MinimumLevel.Verbose()
	.CreateLogger();

var builder = new CommandLineBuilder(root).UseDefaults()
	.UseExceptionHandler((ex, _) => Log.Fatal(ex, "Invocation error"))
	.AddMiddleware(AddDocker.Middleware, MiddlewareOrder.Configuration)
	.AddMiddleware(EnsureBroker.Middleware);

Log.Verbose("Building parser");
var parser = builder.Build();

try {
	Log.Verbose("Invoking parser");
	await parser.InvokeAsync(args);
}
catch (Exception e) {
	Log.Fatal(e, "Unexpected error");
}
finally {
	await Log.CloseAndFlushAsync();
}

return;

static LogEventLevel ConsoleLogLevel() {
	if (Config.VerboseEnabled)
		return LogEventLevel.Verbose;

	// ReSharper disable once ConvertIfStatementToReturnStatement
	if (Config.DebugEnabled)
		return LogEventLevel.Debug;

	return LogEventLevel.Error;
}
