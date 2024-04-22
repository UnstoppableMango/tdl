using System.CommandLine;
using System.CommandLine.Builder;
using System.CommandLine.Parsing;
using UnMango.Tdl.Cli;

var root = new RootCommand("UnstoppableMango's Type Description Language CLI")
{
	Commands.To(),
	Commands.From(),
};

var builder = new CommandLineBuilder(root).UseDefaults();
var parser = builder.Build();

await parser.InvokeAsync(args);
