using Tdl.Plugin.V1;

namespace Tdl.Plugin.Tests;

public class ConnTests
{
	[Fact]
	public void PrefixesTheLength()
	{
		var m = new Handshake { FramingVersion = 1, IrVersion = "tdl.ir.v1" };

		var framed = Repo.Framed(m);

		var body = Google.Protobuf.MessageExtensions.ToByteArray(m);
		Assert.Equal([(byte)body.Length, .. body], framed);
	}

	[Fact]
	public void WritesALongLengthAsAVarint()
	{
		var m = new V1.File { Path = new string('a', 300) };

		var framed = Repo.Framed(m);

		// The body is 303 bytes: a tag, a two-byte length, and the path.
		Assert.Equal(new byte[] { 0xaf, 0x02 }, framed[..2]);
		Assert.Equal(m, new Conn(new MemoryStream(framed), Stream.Null).Recv(V1.File.Parser));
	}

	[Fact]
	public void ReadsMessagesInOrder()
	{
		var a = new Handshake { FramingVersion = 1 };
		var b = new Handshake { IrVersion = "x" };
		var conn = new Conn(new MemoryStream(Repo.Framed(a, b)), Stream.Null);

		Assert.Equal(a, conn.Recv(Handshake.Parser));
		Assert.Equal(b, conn.Recv(Handshake.Parser));
		Assert.Null(conn.Recv(Handshake.Parser));
	}

	[Fact]
	public void AnEmptyMessageIsAZeroLength()
	{
		var conn = new Conn(new MemoryStream([0]), Stream.Null);

		Assert.Equal(new Handshake(), conn.Recv(Handshake.Parser));
	}

	[Fact]
	public void EndingInsideAMessageIsAnError()
	{
		var framed = Repo.Framed(new Handshake { IrVersion = "tdl.ir.v1" });
		var conn = new Conn(new MemoryStream(framed[..^1]), Stream.Null);

		Assert.Throws<EndOfStreamException>(() => conn.Recv(Handshake.Parser));
	}

	[Fact]
	public void EndingInsideALengthIsAnError()
	{
		var conn = new Conn(new MemoryStream([0x80]), Stream.Null);

		Assert.Throws<EndOfStreamException>(() => conn.Recv(Handshake.Parser));
	}

	[Fact]
	public void RefusesALengthOverTheMaximum()
	{
		// 64 MiB and one byte, as a varint: 0x81 0x80 0x80 0x20.
		var conn = new Conn(new MemoryStream([0x81, 0x80, 0x80, 0x20]), Stream.Null);

		var e = Assert.Throws<PluginException>(() => conn.Recv(Handshake.Parser));
		Assert.Contains("maximum size", e.Message);
	}

	[Fact]
	public void RefusesAnOverlongVarint()
	{
		var conn = new Conn(new MemoryStream(Enumerable.Repeat((byte)0xff, 11).ToArray()), Stream.Null);

		Assert.Throws<PluginException>(() => conn.Recv(Handshake.Parser));
	}

	[Fact]
	public void RefusesABodyThatDoesNotParse()
	{
		var conn = new Conn(new MemoryStream([2, 0xff, 0xff]), Stream.Null);

		Assert.Throws<PluginException>(() => conn.Recv(Handshake.Parser));
	}
}
