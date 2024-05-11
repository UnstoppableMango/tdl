using System.CommandLine;
using Docker.DotNet.Models;

namespace UnMango.Tdl.Cli.Internal;

internal interface IDockerProgress : IProgress<string>, IProgress<JSONMessage>;

