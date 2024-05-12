namespace UnMango.Tdl.Cli.Broker;

public sealed record BrokerStatus
{
	public required string State { get; init; }

	public required string Version { get; init; }
}

public interface IBroker
{
	ValueTask<bool> Running(CancellationToken cancellationToken = default);

	ValueTask Start(CancellationToken cancellationToken = default);

	Task<BrokerStatus> Status(CancellationToken cancellationToken = default);

	ValueTask Stop(CancellationToken cancellationToken = default);

	Task Upgrade(string? version, CancellationToken cancellationToken = default);
}

public static class BrokerExtensions
{
	public static Task Upgrade(this IBroker broker, CancellationToken cancellationToken = default)
		=> broker.Upgrade(null, cancellationToken);
}
