using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Docker;

internal interface IDockerProgress : IProgress<string>, IProgress<JSONMessage>;

