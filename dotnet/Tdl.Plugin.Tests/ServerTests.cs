using Tdl.Plugin.Debug;
using Tdl.Plugin.V1;

namespace Tdl.Plugin.Tests;

public class ServerTests
{
	static readonly Handshake Hello = new() { FramingVersion = Conn.FramingVersion, IrVersion = Conn.IrVersion };

	// Serve runs a session over the messages and returns what the plugin
	// wrote, read back, and what it threw.
	static (List<HandshakeReply> Replies, Exception? Error) Serve(IBackend backend, params Google.Protobuf.IMessage[] messages)
	{
		var output = new MemoryStream();
		Exception? error = null;
		try
		{
			Server.Serve(backend, new Conn(new MemoryStream(Repo.Framed(messages)), output));
		}
		catch (Exception e)
		{
			error = e;
		}

		output.Position = 0;
		var conn = new Conn(output, Stream.Null);
		var replies = new List<HandshakeReply>();
		if (conn.Recv(HandshakeReply.Parser) is { } reply)
		{
			replies.Add(reply);
		}
		return (replies, error);
	}

	[Fact]
	public void RepliesWithTheDescription()
	{
		var (replies, error) = Serve(new DebugBackend(), Hello);

		Assert.Null(error);
		var reply = Assert.Single(replies);
		Assert.True(reply.Accepted);
		Assert.Equal(DebugBackend.Name, reply.Name);
		Assert.True(reply.Features.Reuse);
		Assert.False(reply.Features.Reverse);
		Assert.Equal("note", Assert.Single(reply.Directives).Name);
	}

	[Fact]
	public void RefusesAnotherFraming()
	{
		var (replies, error) = Serve(new DebugBackend(), new Handshake { FramingVersion = 2, IrVersion = Conn.IrVersion });

		var reply = Assert.Single(replies);
		Assert.False(reply.Accepted);
		Assert.Equal("framing version 2 is not supported; this plugin speaks 1", reply.Refusal);
		Assert.IsType<PluginException>(error);
	}

	[Fact]
	public void RefusesAnotherModelVersion()
	{
		var (replies, error) = Serve(new DebugBackend(), new Handshake { FramingVersion = 1, IrVersion = "tdl.ir.v2" });

		var reply = Assert.Single(replies);
		Assert.False(reply.Accepted);
		Assert.Equal("model version \"tdl.ir.v2\" is not supported; this plugin reads \"tdl.ir.v1\"", reply.Refusal);
		Assert.NotNull(error);
	}

	[Fact]
	public void RefusesImportWhenItGeneratesOnly()
	{
		var (replies, error) = Serve(new DebugBackend(), new Handshake(Hello) { Mode = Mode.Import });

		var reply = Assert.Single(replies);
		Assert.False(reply.Accepted);
		Assert.Equal("dotnet_debug does not import; it generates only", reply.Refusal);
		Assert.NotNull(error);
	}

	[Fact]
	public void AnswersEveryRequest()
	{
		var req = Repo.Request("shop");
		var output = new MemoryStream();

		Server.Serve(new DebugBackend(), new Conn(new MemoryStream(Repo.Framed(Hello, req, req)), output));

		output.Position = 0;
		var conn = new Conn(output, Stream.Null);
		Assert.NotNull(conn.Recv(HandshakeReply.Parser));
		Assert.NotNull(conn.Recv(Response.Parser));
		Assert.NotNull(conn.Recv(Response.Parser));
		Assert.Null(conn.Recv(Response.Parser));
	}

	[Fact]
	public void ServesImports()
	{
		var output = new MemoryStream();
		var req = new ImportRequest();
		req.Files.Add(new V1.File { Path = "a.txt" });

		Server.Serve(new Echo(), new Conn(new MemoryStream(Repo.Framed(new Handshake(Hello) { Mode = Mode.Import }, req)), output));

		output.Position = 0;
		var conn = new Conn(output, Stream.Null);
		Assert.True(conn.Recv(HandshakeReply.Parser)!.Features.Reverse);
		Assert.Equal("package a.txt\n", conn.Recv(ImportResponse.Parser)!.Source);
	}

	[Fact]
	public void EndingBeforeTheHandshakeIsAnError()
	{
		Assert.Throws<EndOfStreamException>(() => Server.Serve(new DebugBackend(), new Conn(new MemoryStream(), Stream.Null)));
	}

	// Echo imports a file as a package named for its path.
	sealed class Echo : IImporter
	{
		public Description Describe() => new("echo", "0") { Reverse = true };

		public Response Generate(Request request) => new();

		public ImportResponse Import(ImportRequest request) =>
			new() { Source = $"package {request.Files[0].Path}\n" };
	}
}
