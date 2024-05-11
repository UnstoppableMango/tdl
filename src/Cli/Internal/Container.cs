using Docker.DotNet.Models;
using Serilog;

namespace UnMango.Tdl.Cli.Internal;

public interface IContainer : IAsyncDisposable
{
	string Id { get; }
}

internal sealed class Container(IDocker docker, CreateContainerResponse create) : IContainer
{
	public string Id => create.ID;

	public async ValueTask DisposeAsync() {
		Log.Verbose("Disposing container");
		await docker.Stop(this);
	}
}
