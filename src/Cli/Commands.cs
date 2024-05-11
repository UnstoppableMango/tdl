using System.CommandLine;
using System.CommandLine.Binding;

namespace UnMango.Tdl.Cli;

file class TokenBinder : BinderBase<CancellationToken>
{
	public static readonly TokenBinder Value = new();

	protected override CancellationToken GetBoundValue(BindingContext bindingContext) {
		var token = bindingContext.GetService(typeof(CancellationToken));
		if (token is null) throw new Exception("Unable to retrieve cancellationToken");
		return (CancellationToken)token;
	}
}

internal static class Commands
{
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
