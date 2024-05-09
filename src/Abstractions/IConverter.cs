namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public interface IConverter
{
	Task<Spec> FromAsync(Stream input, CancellationToken cancellationToken = default);
}
