using System.CommandLine;
using Docker.DotNet;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class Docker(IConsole console, IDockerClient docker)
{
	public async Task RunPlugin(
		string plugin,
		Stream input,
		Stream output,
		CancellationToken cancellationToken = default) {
		const string tag = "main";
		var image = $"ghcr.io/unstoppablemango/{plugin}";
		var progress = new ConsoleProgress(console);

		await docker.Images.CreateImageAsync(
			new ImagesCreateParameters { FromImage = image, Tag = tag },
			null,
			progress,
			cancellationToken);

		var container = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{image}:{tag}",
				Name = "tdl-test",
				StdinOnce = true,
				OpenStdin = true,
				AttachStdin = true,
				AttachStdout = true,
				Cmd = ["gen"],
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
