using System.CommandLine;
using System.CommandLine.Builder;
using System.CommandLine.Invocation;
using System.CommandLine.Parsing;
using Serilog;
using UnMango.Tdl.Cli;
using UnMango.Tdl.Cli.Internal;

var root = new RootCommand("UnstoppableMango's Type Description Language CLI") {
	Commands.Gen(),
	Commands.From(),
};

Log.Logger = new LoggerConfiguration()
	.Enrich.FromLogContext()
	.WriteTo.Console()
	.MinimumLevel.Verbose()
	.CreateLogger();

var builder = new CommandLineBuilder(root)
	.UseDefaults()
	.UseExceptionHandler((exception, _) => {
		Log.Fatal(exception, "Invocation error");
	})
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
