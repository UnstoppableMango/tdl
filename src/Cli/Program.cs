using System.CommandLine;
using System.CommandLine.Builder;
using System.CommandLine.Parsing;
using Serilog;
using UnMango.Tdl;
using UnMango.Tdl.Cli;

var root = new RootCommand("UnstoppableMango's Type Description Language CLI") {
	Commands.To(),
	Commands.Gen(),
	Commands.From(),
};

Log.Logger = new LoggerConfiguration()
	.Enrich.FromLogContext()
	.WriteTo.Console()
	.CreateLogger();

var builder = new CommandLineBuilder(root)
	.AddMiddleware(async (context, next) => {
		if (Environment.GetEnvironmentVariable("ENABLE_BROKER") == "true") {
			var cancellationToken = context.GetCancellationToken();
			using var scope = await Broker.Dev.Start("http://127.0.0.1:6969", cancellationToken);
		}

		await next(context);
	})
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
