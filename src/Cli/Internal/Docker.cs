using Docker.DotNet;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

public sealed record StartArgs
{
	public required string Image { get; init; }
	public required string Tag { get; init; }
	public IList<string> Cmd { get; init; } = [];
	public string? Name { get; init; }
	public IList<string> Volumes { get; init; } = [];
}

public sealed record StartResult(CreateContainerResponse Container);

public interface IDocker
{
	Task<StartResult> Start(StartArgs args, CancellationToken cancellationToken = default);

	Task Stop(string id, CancellationToken cancellationToken = default);
}

internal static class DockerExtensions
{
	public static Task Stop(this IDocker docker, StartResult start, CancellationToken cancellationToken = default)
		=> docker.Stop(start.Container.ID, cancellationToken);
}

internal sealed class Docker(IDockerClient docker, IDockerProgress progress) : IDocker
{
	private static readonly Random Random = new();
	private static string RandomName => $"tdl-{Random.Next()}";

	public async Task<StartResult> Start(StartArgs args, CancellationToken cancellationToken) {
		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters {
				FromImage = args.Image,
				Tag = args.Tag,
			},
			new AuthConfig(),
			progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{args.Image}:{args.Tag}",
				Name = args.Name ?? RandomName,
				Cmd = args.Cmd,
				HostConfig = new HostConfig {
					Binds = args.Volumes,
				},
			},
			cancellationToken);

		// TODO: This isn't working the way I want it to. i.e. nothing is output
		await docker.Containers.GetContainerLogsAsync(
			container.ID,
			new ContainerLogsParameters {
				Follow = true,
				ShowStdout = true,
				ShowStderr = true,
			},
			cancellationToken,
			progress);

		var started = await docker.Containers.StartContainerAsync(
			container.ID,
			new ContainerStartParameters(),
			cancellationToken);

		if (!started) {
			throw new Exception("Failed to start the container.");
		}

		return new StartResult(container);
	}

	public async Task Stop(string id, CancellationToken cancellationToken) {
		_ = await docker.Containers.StopContainerAsync(
			id,
			new ContainerStopParameters {
				WaitBeforeKillSeconds = 15,
			},
			cancellationToken);

		await docker.Containers.RemoveContainerAsync(
			id,
			new ContainerRemoveParameters(),
			cancellationToken);
	}
}
