using Docker.DotNet.Models;
using Serilog;

namespace UnMango.Tdl.Cli.Internal;

internal sealed class SerilogProgress : IDockerProgress
{
	private readonly ILogger _logger = Log.ForContext<IDocker>();

	public void Report(string value) {
		_logger.Debug("{Message}", value);
	}

	public void Report(JSONMessage value) {
		_logger.Debug("{Message}", value.Status);
	}
}
