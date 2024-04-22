using System.CommandLine;

namespace UnMango.Tdl.Cli;

internal static class Commands
{
	public static Command From() {
		var sourceArg = new Argument<string>("source", "The source format.");
		var toOption = new Option<string?>(["--to", "-t", "-2"], "The target format.");
		var fileArg = new Argument<IEnumerable<FileInfo>>("file", "The file(s) to convert.") {
			Arity = ArgumentArity.ZeroOrMore,
		};
		var command = new Command("from", "Convert the source format to uml.") {
			sourceArg, toOption, fileArg,
		};
		command.SetHandler(Handlers.From, sourceArg, toOption, fileArg);
		return command;
	}

	public static Command To() {
		var targetArg = new Argument<string>("target", "The target output format.");
		var fromOption = new Option<string?>(["--from", "-f"], "The source format.");
		var fileArg = new Argument<IEnumerable<FileInfo>>("file", "The file(s) to convert.") {
			Arity = ArgumentArity.ZeroOrMore,
		};
		var command = new Command("to", "Convert from uml to the target output format.") {
			targetArg, fromOption, fileArg,
		};
		command.AddAlias("2");
		command.SetHandler(Handlers.To, targetArg, fromOption, fileArg);
		return command;
	}
}
