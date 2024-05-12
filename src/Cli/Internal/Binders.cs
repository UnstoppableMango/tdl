using System.CommandLine;
using System.CommandLine.Binding;
using Microsoft.Extensions.DependencyInjection;

namespace UnMango.Tdl.Cli.Internal;

internal static class Binder
{
	public static BinderBase<T> Service<T>() where T : class
		=> ServiceProvider<T>.Value;

	private sealed class ServiceProvider<T> : BinderBase<T> where T : class
	{
		public static readonly ServiceProvider<T> Value = new();

		protected override T GetBoundValue(BindingContext bindingContext) {
			return bindingContext.GetRequiredService<T>();
		}
	}
}

internal sealed class TokenBinder : BinderBase<CancellationToken>
{
	public static readonly TokenBinder Value = new();

	protected override CancellationToken GetBoundValue(BindingContext bindingContext) {
		return bindingContext.GetRequiredService<CancellationToken>();
	}
}

internal sealed class ConsoleBinder : BinderBase<IConsole>
{
	public static readonly ConsoleBinder Value = new();

	protected override IConsole GetBoundValue(BindingContext bindingContext) {
		return bindingContext.Console;
	}
}
