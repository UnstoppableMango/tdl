using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class DockerRunner(IDocker docker, Config config, string plugin) : IRunner
{
	private readonly StartArgs _defaultArgs = new() {
		Image = $"{config.ContainerRepo}/{plugin}",
		Tag = config.ContainerTag,
		Volumes = [$"{config.SocketDir}:/var/run/tdl"],
	};

	public async Task<Spec> FromAsync(Stream input, CancellationToken cancellationToken = default) {
		var container = await docker.Start(_defaultArgs with { Cmd = ["from"] }, cancellationToken);
		await docker.Stop(container, cancellationToken);
		return new Spec();
	}

	public async Task GenerateAsync(Spec spec, Stream output, CancellationToken cancellationToken = default) {
		var container = await docker.Start(_defaultArgs with { Cmd = ["gen"] }, cancellationToken);
		await docker.Stop(container, cancellationToken);
	}
}
