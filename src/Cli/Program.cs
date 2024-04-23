using System.CommandLine;
using System.CommandLine.Builder;
using System.CommandLine.Parsing;
using UnMango.Tdl;
using UnMango.Tdl.Cli;

var root = new RootCommand("UnstoppableMango's Type Description Language CLI") {
	Commands.To(),
	Commands.Gen(),
	Commands.From(),
};

var builder = new CommandLineBuilder(root)
	.AddMiddleware(async (context, next) => {
		var cancellationToken = context.GetCancellationToken();
		var scope = await Broker.Dev.Start("http://127.0.0.1:6969", cancellationToken);
		await next(context);
		await scope.DisposeAsync();
	})
	.UseDefaults();

var parser = builder.Build();

await parser.InvokeAsync(args);
