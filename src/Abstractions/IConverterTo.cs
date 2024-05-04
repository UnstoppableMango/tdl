namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public interface IConverterTo
{
	Task ToAsync(Stream input, Stream output, CancellationToken cancellationToken = default);
}
