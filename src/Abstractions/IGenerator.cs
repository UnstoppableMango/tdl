namespace UnMango.Tdl.Abstractions;

[PublicAPI]
public interface IGenerator
{
	Task GenerateAsync(Spec input, Stream output, CancellationToken cancellationToken = default);
}
