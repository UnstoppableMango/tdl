using Docker.DotNet;
using Docker.DotNet.Models;
using Serilog;

namespace UnMango.Tdl.Cli.Docker;

internal sealed class Docker(IDockerClient docker, IDockerProgress progress) : IDocker
{
	private static readonly Random Random = new();
	private static string RandomName => $"tdl-{Random.Next()}";

	public async Task<IContainer?> FindMatching(
		IDictionary<string, string> labels,
		CancellationToken cancellationToken) {
		Dictionary<string, IDictionary<string, bool>> filters = new() {
			["label"] = labels.ToDictionary(
				x => $"{x.Key}={x.Value}",
				_ => true),
		};

		Log.Debug("Listing containers");
		var containers = await docker.Containers.ListContainersAsync(
			new ContainersListParameters {
				Filters = filters,
			},
			cancellationToken);

		var firstMatch = containers.FirstOrDefault();
		if (firstMatch is not null)
			return Container.From(this, firstMatch);

		Log.Debug("No match found");
		return null;
	}

	public Task FollowLogs(string id, CancellationToken cancellationToken) {
		Log.Debug("Getting container logs");
		return docker.Containers.GetContainerLogsAsync(
			id,
			new ContainerLogsParameters {
				Follow = true,
				ShowStdout = true,
				ShowStderr = true,
			},
			cancellationToken,
			progress);
	}

	public async Task<InspectResult> Inspect(string id, CancellationToken cancellationToken) {
		var result = await docker.Containers.InspectContainerAsync(id, cancellationToken);

		return new InspectResult {
			Version = result.Config.Image.Split(':')[1],
			State = result.State.Status,
		};
	}

	public async Task<IContainer> Start(StartArgs args, CancellationToken cancellationToken) {
		if (Config.Env.IsRelease) {
			Log.Debug("Creating image");
			await docker.Images.CreateImageAsync(
				new ImagesCreateParameters {
					FromImage = args.Image,
					Tag = args.Tag,
				},
				new AuthConfig(),
				progress,
				cancellationToken);
		}

		Log.Debug("Creating container");
		var createResponse = await docker.Containers.CreateContainerAsync(
			new CreateContainerParameters {
				Image = $"{args.Image}:{args.Tag}",
				Name = args.Name ?? RandomName,
				Cmd = args.Cmd,
				User = args.User,
				Tty = true,
				Labels = args.Labels,
				HostConfig = new HostConfig {
					Binds = args.Volumes,
					Mounts = args.Tmpfs.Select(x => new Mount {
						Type = "tmpfs",
						Target = x,
					}).ToList(),
				},
			},
			cancellationToken);

		Log.Debug("Starting container");
		var started = await docker.Containers.StartContainerAsync(
			createResponse.ID,
			new ContainerStartParameters(),
			cancellationToken);

		if (!started) {
			throw new Exception("Failed to start the container.");
		}

		Log.Verbose("Started container");
		return Container.From(this, createResponse);
	}

	public async Task Stop(string id, CancellationToken cancellationToken) {
		Log.Debug("Stopping container");
		_ = await docker.Containers.StopContainerAsync(
			id,
			new ContainerStopParameters {
				WaitBeforeKillSeconds = 15,
			},
			cancellationToken);

		Log.Debug("Removing container");
		await docker.Containers.RemoveContainerAsync(
			id,
			new ContainerRemoveParameters(),
			cancellationToken);
	}

	public Task WaitFor(string id, Predicate<string> condition, CancellationToken cancellationToken) {
		var tcs = new TaskCompletionSource();
		var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
		cancellationToken.Register(() => tcs.SetCanceled(cancellationToken));

		var subject = new Subject<string>(condition, () => {
			Log.Debug("Condition met");
			cts.Cancel();
			tcs.SetResult();
		});

		Log.Debug("Getting container logs");
		_ = docker.Containers.GetContainerLogsAsync(
			id,
			new ContainerLogsParameters {
				ShowStderr = true,
				ShowStdout = true,
				Follow = true,
			},
			cts.Token,
			subject);

		Log.Debug("Waiting for condition");
		return tcs.Task;
	}

	private sealed class Subject<T>(Predicate<T> condition, Action onComplete) : IProgress<T>
	{
		public void Report(T value) {
			if (condition(value)) onComplete();
		}
	}
}
