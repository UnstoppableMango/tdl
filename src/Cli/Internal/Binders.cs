using System.CommandLine.Binding;
using Microsoft.Extensions.DependencyInjection;
using UnMango.Tdl.Cli.Docker;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class TokenBinder : BinderBase<CancellationToken>
{
	public static readonly TokenBinder Value = new();

	protected override CancellationToken GetBoundValue(BindingContext bindingContext) {
		return bindingContext.GetRequiredService<CancellationToken>();
	}
}

internal class DockerBinder : BinderBase<IDocker>
{
	public static readonly DockerBinder Value = new();

	protected override IDocker GetBoundValue(BindingContext bindingContext) {
		return bindingContext.GetRequiredService<IDocker>();
	}
}
