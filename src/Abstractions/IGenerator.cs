namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public interface IGenerator
{
	Task GenerateAsync(Stream input, Stream output, CancellationToken cancellationToken = default);
}
