namespace UnMango.Tdl.Cli.Broker;

public sealed record BrokerStatus
{
	public required string State { get; init; }

	public required string Version { get; init; }
}

public interface IBroker
{
	ValueTask<bool> Running(CancellationToken cancellationToken);

	ValueTask Start(CancellationToken cancellationToken);

	Task<BrokerStatus> Status(CancellationToken cancellationToken);

	ValueTask Stop(CancellationToken cancellationToken);

	Task Upgrade(string version, CancellationToken cancellationToken);
}
