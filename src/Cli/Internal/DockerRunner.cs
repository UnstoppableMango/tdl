using System.CommandLine;
using Docker.DotNet;
using Docker.DotNet.Models;
using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class DockerRunner(IConsole console, IDockerClient docker, string plugin) : IRunner
{
	private const string Tag = "main", ContainerName = "tdl-gen", Port = "42069/tcp";
	private readonly string _image = $"ghcr.io/unstoppablemango/{plugin}";
	private readonly ConsoleProgress _progress = new(console);

	public async Task<Spec> FromAsync(Stream input, CancellationToken cancellationToken = default) {
		var result = await Start(["from"], cancellationToken);
		await Stop(result.Container.ID, cancellationToken);

		throw new NotImplementedException();
	}

	public async Task GenerateAsync(Spec spec, Stream output, CancellationToken cancellationToken = default) {
		var result = await Start(["gen"], cancellationToken);
		await Stop(result.Container.ID, cancellationToken);
	}

	private async Task<StartResult> Start(IList<string> command, CancellationToken cancellationToken) {
		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters { FromImage = _image, Tag = Tag },
			null,
			_progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{_image}:{Tag}",
				Name = ContainerName,
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
			_progress);

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

	private async Task Stop(string id, CancellationToken cancellationToken) {
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

	private sealed record ConsoleProgress(IConsole Console) : IProgress<JSONMessage>, IProgress<string>
	{
		public void Report(JSONMessage value) {
			// Console.Write("Got progress: ");
			// Console.WriteLine(value.ProgressMessage);
			// Console.WriteLine(value.ErrorMessage);
			Console.Write(value.Status);
		}

		public void Report(string value) => Console.WriteLine(value);
	}

	private sealed record StartResult(CreateContainerResponse Container, ContainerInspectResponse Inspection);
}
