namespace UnMango.Tdl.Cli.Docker;

public sealed class InspectResult
{
	public required string State { get; init; }

	public required string Version { get; init; }
}

public sealed record StartArgs
{
	public required string Image { get; init; }
	public required string Tag { get; init; }
	public IList<string> Cmd { get; init; } = [];
	public string? Name { get; init; }
	public IList<string> Volumes { get; init; } = [];
	public string? User { get; init; }
	public IDictionary<string, string> Labels { get; init; } = new Dictionary<string, string>();
}

public interface IDocker
{
	Task<IContainer?> FindMatching(IDictionary<string, string> labels, CancellationToken cancellationToken = default);

	Task FollowLogs(string id, CancellationToken cancellationToken = default);

	Task<InspectResult> Inspect(string id, CancellationToken cancellationToken = default);

	Task<IContainer> Start(StartArgs args, CancellationToken cancellationToken = default);

	Task Stop(string id, CancellationToken cancellationToken = default);

	Task WaitFor(string id, Predicate<string> condition, CancellationToken cancellationToken = default);
}

internal static class DockerExtensions
{
	public static Task FollowLogs(
		this IDocker docker,
		IContainer container,
		CancellationToken cancellationToken = default)
		=> docker.FollowLogs(container.Id, cancellationToken);

	public static Task<InspectResult> Inspect(
		this IDocker docker,
		IContainer container,
		CancellationToken cancellationToken = default)
		=> docker.Inspect(container.Id, cancellationToken);

	public static Task Stop(this IDocker docker, IContainer container, CancellationToken cancellationToken = default)
		=> docker.Stop(container.Id, cancellationToken);

	public static Task WaitFor(
		this IDocker docker,
		IContainer container,
		Predicate<string> condition,
		CancellationToken cancellationToken = default)
		=> docker.WaitFor(container.Id, condition, cancellationToken);
}
