using System.CommandLine;
using Docker.DotNet;
using Docker.DotNet.Models;
using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class DockerRunner(IDocker docker, string plugin) : IRunner
{
	private const string Tag = "main";
	private readonly string _image = $"ghcr.io/unstoppablemango/{plugin}";

	public async Task<Spec> FromAsync(Stream input, CancellationToken cancellationToken = default) {
		var result = await docker.Start(_image, Tag, ["from"], cancellationToken);
		await docker.Stop(result.Container.ID, cancellationToken);

		throw new NotImplementedException();
	}

	public async Task GenerateAsync(Spec spec, Stream output, CancellationToken cancellationToken = default) {
		var result = await docker.Start(_image, Tag, ["gen"], cancellationToken);
		await docker.Stop(result.Container.ID, cancellationToken);
	}
}
