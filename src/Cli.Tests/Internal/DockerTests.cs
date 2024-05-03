using System.Text;
using Docker.DotNet;
using Google.Protobuf;
using Xunit.Abstractions;

namespace UnMango.Tdl.Cli.Tests.Internal;

public sealed class DockerTests(ITestOutputHelper test)
{
	[Fact]
	public async Task TestDocker() {
		using var client = new DockerClientConfiguration()
			.CreateClient();

		var docker = new Cli.Internal.Docker(new XUnitConsole(test), client, "uml2ts");
		var spec = new Spec {
			Name = "test-spec",
			Types_ = {
				["testType"] = new Type {
					Type_ = "object",
				},
			},
		};

		await using MemoryStream input = new(spec.ToByteArray()), output = new();
		Assert.True(input.Length > 0, $"input length was: {input.Length}");

		await docker.GenerateAsync(input, output);

		Assert.True(output.Length > 0, $"output length was: {output.Length}");
		var actual = Encoding.UTF8.GetString(output.ToArray());
		test.WriteLine(actual);
		Assert.Equal("export interface testType \n{\n}", actual);
	}
}
