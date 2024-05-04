namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public interface IConverterFrom
{
	Task FromAsync(Stream input, Stream output, CancellationToken cancellationToken = default);
}
