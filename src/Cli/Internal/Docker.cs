using Docker.DotNet;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

public sealed record StartResult(CreateContainerResponse Container, ContainerInspectResponse Inspection);

public interface IDocker
{
	Task<StartResult> Start(
		string image,
		string tag,
		IList<string> command,
		string name,
		CancellationToken cancellationToken = default);

	Task Stop(string id, CancellationToken cancellationToken = default);
}

internal static class DockerExtensions
{
	private static readonly Random Random = new();
	private static string RandomName => $"tdl-{Random.Next()}";

	public static Task<StartResult> Start(
		this IDocker docker,
		string image,
		string tag,
		IList<string> command,
		CancellationToken cancellationToken)
		=> docker.Start(image, tag, command, RandomName, cancellationToken);

	public static Task Stop(this IDocker docker, StartResult start, CancellationToken cancellationToken = default)
		=> docker.Stop(start.Container.ID, cancellationToken);
}

internal sealed class Docker(IDockerClient docker, IDockerProgress progress) : IDocker
{
	private const string Port = "42069/tcp";

	public async Task<StartResult> Start(
		string image,
		string tag,
		IList<string> command,
		string name,
		CancellationToken cancellationToken) {
		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters { FromImage = image, Tag = tag },
			null,
			progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{image}:{tag}",
				Name = name,
				AttachStdout = true,
				AttachStderr = true,
				Cmd = command,
				ExposedPorts = { [Port] = new EmptyStruct() },
			},
			cancellationToken);


		await docker.Containers.GetContainerLogsAsync(
			container.ID,
			new ContainerLogsParameters { Follow = true, ShowStdout = true },
			cancellationToken,
			progress);

		var started = await docker.Containers.StartContainerAsync(
			container.ID,
			new ContainerStartParameters(),
			cancellationToken);

		if (!started) {
			throw new Exception("Failed to start the container.");
		}

		var inspect = await docker.Containers.InspectContainerAsync(
			container.ID,
			cancellationToken);

		return new StartResult(container, inspect);
	}

	public async Task Stop(string id, CancellationToken cancellationToken) {
		await docker.Containers.StopContainerAsync(
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
