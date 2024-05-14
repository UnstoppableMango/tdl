namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public class Runner(IConverter converter, IGenerator generator) : IRunner
{
	public virtual Task<Spec> FromAsync(Stream input, CancellationToken cancellationToken = default) {
		return converter.FromAsync(input, cancellationToken);
	}

	public virtual Task GenerateAsync(Spec spec, Stream output, CancellationToken cancellationToken = default) {
		return generator.GenerateAsync(spec, output, cancellationToken);
	}

	public Runner With(IConverter next) => new(next, generator);
	public Runner With(IGenerator next) => new(converter, next);
}

[PublicAPI]
public static class RunnerExtensions
{
	public static IRunner With(this IRunner runner, IConverter converter) => new Runner(converter, runner);
	public static IRunner With(this IRunner runner, IGenerator generator) => new Runner(runner, generator);
}
