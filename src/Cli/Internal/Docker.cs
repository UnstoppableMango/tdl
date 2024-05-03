using System.CommandLine;
using Docker.DotNet;
using Docker.DotNet.Models;
using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class Docker(IConsole console, IDockerClient docker, string plugin) : IConverter, IGenerator
{
	private const string Tag = "main";
	private readonly string _image = $"ghcr.io/unstoppablemango/{plugin}";

	public Task ToAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		throw new NotImplementedException();
	}

	public Task FromAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		throw new NotImplementedException();
	}

	public Task GenerateAsync(Stream input, Stream output, CancellationToken cancellationToken = default) {
		return Run("tdl-gen-test", ["gen"], input, output, cancellationToken);
	}

	private async Task Run(
		string name,
		IList<string> command,
		Stream input,
		Stream output,
		CancellationToken cancellationToken = default) {
		var progress = new ConsoleProgress(console);

		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters { FromImage = _image, Tag = Tag },
			null,
			progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{_image}:{Tag}",
				Name = name,
				StdinOnce = true,
				OpenStdin = true,
				AttachStdin = true,
				AttachStdout = true,
				Cmd = command,
			},
			cancellationToken);

		using var stream = await docker.Containers.AttachContainerAsync(
			container.ID,
			false,
			new ContainerAttachParameters {
				Stdin = true,
				Stdout = true,
				Stderr = true,
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
			await stream.CopyFromAsync(input, cancellationToken);
			await stream.CopyOutputToAsync(null, output, null, cancellationToken);
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

	private static string ImageFor(string plugin) => $"ghcr.io/unstoppablemango/{plugin}";

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
