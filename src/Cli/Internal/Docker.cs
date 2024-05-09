using System.CommandLine;
using Docker.DotNet;
using Docker.DotNet.Models;
using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class Docker(IConsole console, IDockerClient docker, string plugin) : IConverter, IGenerator
{
	private const string Tag = "main", ContainerName = "tdl-gen";
	private readonly string _image = $"ghcr.io/unstoppablemango/{plugin}";

	public Task ToAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		throw new NotImplementedException();
	}

	public Task FromAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		throw new NotImplementedException();
	}

	public Task GenerateAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		return Run(["gen"], input, output, cancellationToken);
	}

	private async Task Run(IList<string> command, CancellationToken cancellationToken = default) {
		var progress = new ConsoleProgress(console);

		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters { FromImage = _image, Tag = Tag },
			null,
			progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{_image}:{Tag}",
				Name = ContainerName,
				AttachStdout = true,
				AttachStderr = true,
				Cmd = command,
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

		try {
		}
		finally {
			await docker.Containers.StopContainerAsync(
				container.ID,
				new ContainerStopParameters {
					WaitBeforeKillSeconds = 15,
				},
				cancellationToken);

			await docker.Containers.RemoveContainerAsync(
				container.ID,
				new ContainerRemoveParameters(),
				cancellationToken);
		}
	}

	private sealed record ConsoleProgress(IConsole Console) : IProgress<JSONMessage>, IProgress<string>
	{
		public void Report(JSONMessage value) {
			// Console.Write("Got progress: ");
			// Console.WriteLine(value.ProgressMessage);
			// Console.WriteLine(value.ErrorMessage);
			Console.Write(value.Status);
		}

		public void Report(string value) {
			Console.WriteLine(value);
		}
	}
}
