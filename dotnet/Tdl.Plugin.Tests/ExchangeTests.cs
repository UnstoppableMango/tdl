using Google.Protobuf;
using Tdl.Plugin.Debug;
using Tdl.Plugin.V1;

namespace Tdl.Plugin.Tests;

// The debug port answers each request the Go host recorded in
// testdata/plugin with the bytes the Go debug backend answered it with.
public class ExchangeTests
{
	public static TheoryData<string> Exchanges => Repo.Exchanges();

	[Fact]
	public void ThereAreExchanges() => Assert.NotEmpty(Exchanges);

	[Theory]
	[MemberData(nameof(Exchanges))]
	public void AnswersByteForByte(string exchange)
	{
		var resp = new DebugBackend().Generate(Repo.Request(exchange));

		Assert.Equal(Repo.Response(exchange), resp.ToByteArray());
	}

	[Theory]
	[MemberData(nameof(Exchanges))]
	public void AnswersOverAConnection(string exchange)
	{
		var hello = new Handshake { FramingVersion = Conn.FramingVersion, IrVersion = Conn.IrVersion };
		var input = new MemoryStream(Repo.Framed(hello, Repo.Request(exchange)));
		var output = new MemoryStream();

		Server.Serve(new DebugBackend(), new Conn(input, output));

		var reply = new DebugBackend().Describe().Reply();
		var want = Repo.Framed(reply).Concat(Repo.Framed(Response.Parser.ParseFrom(Repo.Response(exchange)))).ToArray();
		Assert.Equal(want, output.ToArray());
	}
}
