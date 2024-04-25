using Microsoft.AspNetCore.Hosting.Server;
using Microsoft.AspNetCore.Http.Features;

namespace UnMango.Tdl.Cli;

internal sealed record Context(IFeatureCollection Features);

internal sealed class Application : IHttpApplication<Context>
{
	public Context CreateContext(IFeatureCollection contextFeatures) => new(contextFeatures);

	public Task ProcessRequestAsync(Context context) {
		throw new NotImplementedException();
	}

	public void DisposeContext(Context context, Exception? exception) { }
}
