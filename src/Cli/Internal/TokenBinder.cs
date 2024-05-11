using System.CommandLine.Binding;

namespace UnMango.Tdl.Cli.Internal;

internal class TokenBinder : BinderBase<CancellationToken>
{
	public static readonly TokenBinder Value = new();

	protected override CancellationToken GetBoundValue(BindingContext bindingContext) {
		var token = bindingContext.GetService(typeof(CancellationToken));
		if (token is null) throw new Exception("Unable to retrieve cancellationToken");
		return (CancellationToken)token;
	}
}
