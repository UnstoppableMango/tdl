using Docker.DotNet.Models;
using NSubstitute;
using UnMango.Tdl.Cli.Internal;

namespace UnMango.Tdl.Cli.Tests.Internal;

public sealed class DockerRunnerTests
{
	[Fact]
	public async Task TestDocker() {
		await using MemoryStream output = new();
		var spec = new Spec();
		var createResponse = new CreateContainerResponse { ID = "test-id" };
		var client = Substitute.For<IDocker>();
		var startResult = new Container(client, createResponse);

		client.Start(Arg.Any<StartArgs>(), CancellationToken.None)
			.Returns(startResult);

		var docker = new DockerRunner(client, new Config(Env.Dev), string.Empty);

		await docker.GenerateAsync(spec, output);

		// TODO: Assertions
		List<string> expectedCmd = ["gen"];
		await client.Received().Start(Arg.Any<StartArgs>(), CancellationToken.None);
		await client.Received().Stop(createResponse.ID, CancellationToken.None);
	}
}
