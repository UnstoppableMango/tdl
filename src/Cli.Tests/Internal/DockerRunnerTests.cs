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
		var startResult = new StartResult(createResponse, new ContainerInspectResponse());
		var client = Substitute.For<IDocker>();

		client.Start(
				Arg.Any<string>(),
				Arg.Any<string>(),
				Arg.Any<IList<string>>(),
				Arg.Any<string>(),
				CancellationToken.None)
			.Returns(startResult);

		var docker = new DockerRunner(client, string.Empty);

		await docker.GenerateAsync(spec, output);

		List<string> expectedCmd = ["gen"];
		await client.Received().Start(
			Arg.Any<string>(),
			Arg.Any<string>(),
			Arg.Is<IList<string>>(x => x.SequenceEqual(expectedCmd)),
			Arg.Any<string>(),
			CancellationToken.None);
		await client.Received().Stop(createResponse.ID, CancellationToken.None);
	}
}
