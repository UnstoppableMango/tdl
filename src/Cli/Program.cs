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
	.CreateLogger();

var builder = new CommandLineBuilder(root)
	.AddMiddleware(AddDocker.Middleware, MiddlewareOrder.Configuration)
	.AddMiddleware(EnsureBroker.Middleware)
	.UseDefaults();

var parser = builder.Build();

try {
	await parser.InvokeAsync(args);
}
catch (Exception e) {
	Log.Fatal(e, "Unexpected error");
}
finally {
	await Log.CloseAndFlushAsync();
}
