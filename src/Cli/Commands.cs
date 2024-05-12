using System.CommandLine;
using UnMango.Tdl.Cli.Broker;
using UnMango.Tdl.Cli.Internal;

namespace UnMango.Tdl.Cli;

internal static class Commands
{
	public static Command Broker() {
		return new Command("broker", "Manage the tdl broker") {
			BrokerCommands.Status(),
			BrokerCommands.Start(),
			BrokerCommands.Stop(),
			BrokerCommands.Upgrade(),
		};
	}

	public static Command From() {
		var sourceArg = new Argument<string>("source", "The source format");
		var toOpt = new Option<string?>(["--to", "-t", "-2"], "The target format");
		var fileArg = new Argument<IEnumerable<FileInfo>>("file", "The file(s) to convert") {
			Arity = ArgumentArity.ZeroOrMore,
		};
		var command = new Command("from", "Convert the source format to uml") {
			sourceArg, toOpt, fileArg,
		};
		command.SetHandler(Handlers.From, sourceArg, toOpt, fileArg, TokenBinder.Value);
		return command;
	}

	public static Command Gen() {
		var targetArg = new Argument<string>("target", "The target output format");
		var sourceArg = new Argument<string>("source", "The source format");
		var fileArg = new Argument<IEnumerable<FileInfo>>("file", "The file(s) to convert") {
			Arity = ArgumentArity.ZeroOrMore,
		};
		var command = new Command("generate", "Generate types for the target format") {
			sourceArg, fileArg,
		};
		command.AddAlias("gen");
		command.SetHandler(Handlers.Gen, targetArg, sourceArg, fileArg, TokenBinder.Value);
		return command;
	}
}
