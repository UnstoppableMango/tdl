using Docker.DotNet.Models;
using Serilog;

namespace UnMango.Tdl.Cli.Docker;

public interface IContainer : IAsyncDisposable
{
	string Id { get; }
}

internal sealed class Container(IDocker docker, string id) : IContainer
{
	public string Id => id;

	public async ValueTask DisposeAsync() {
		Log.Verbose("Disposing container");
		await docker.Stop(this);
	}

	public static Container From(IDocker docker, CreateContainerResponse create) => new(docker, create.ID);

	public static Container From(IDocker docker, ContainerListResponse list) => new(docker, list.ID);
}
