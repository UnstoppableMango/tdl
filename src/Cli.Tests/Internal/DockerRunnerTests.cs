using System.Text;
using Docker.DotNet;
using Google.Protobuf;
using NSubstitute;
using UnMango.Tdl.Cli.Internal;
using Xunit.Abstractions;

namespace UnMango.Tdl.Cli.Tests.Internal;

public sealed class DockerRunnerTests(ITestOutputHelper test)
{
	[Fact]
	public async Task TestDocker() {
		var client = Substitute.For<IDocker>();
		var docker = new DockerRunner(client, string.Empty);
		await using MemoryStream output = new();

		var spec = new Spec {
			Name = "test-spec",
			Types_ = {
				["testType"] = new Type {
					Type_ = "object",
				},
			},
		};

		await docker.GenerateAsync(spec, output);

		Assert.True(output.Length > 0, $"output length was: {output.Length}");
		var actual = Encoding.UTF8.GetString(output.ToArray());
		test.WriteLine(actual);
		Assert.Equal("export interface testType \n{\n}", actual);
	}
}
