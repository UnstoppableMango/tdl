using Google.Protobuf;

namespace Tdl.Plugin;

/// <summary>
/// One end of a plugin connection. Each message is framed as a varint byte
/// count followed by the encoded message. It is not safe for concurrent use.
/// </summary>
public sealed class Conn
{
	/// <summary>The wire framing this class implements.</summary>
	public const int FramingVersion = 1;

	/// <summary>The protobuf package the model is encoded with.</summary>
	public const string IrVersion = "tdl.ir.v1";

	/// <summary>
	/// Bounds a single message. A length prefix over it is refused before
	/// anything is allocated.
	/// </summary>
	public const int MaxMessageSize = 64 << 20;

	// A varint holding a 64-bit length is at most ten bytes.
	const int MaxVarintLength = 10;

	readonly Stream input;
	readonly Stream output;

	/// <summary>Returns a connection reading from input and writing to output.</summary>
	public Conn(Stream input, Stream output)
	{
		this.input = input;
		this.output = output;
	}

	/// <summary>Encodes a message and writes it with its length prefix.</summary>
	public void Send(IMessage message)
	{
		var body = message.ToByteArray();
		if (body.Length > MaxMessageSize)
		{
			throw new PluginException($"message exceeds the maximum size: {body.Length} bytes");
		}

		Span<byte> prefix = stackalloc byte[MaxVarintLength];
		var n = 0;
		for (var size = (ulong)body.Length; ; size >>= 7)
		{
			if (size < 0x80)
			{
				prefix[n++] = (byte)size;
				break;
			}
			prefix[n++] = (byte)(size | 0x80);
		}

		output.Write(prefix[..n]);
		output.Write(body);
		output.Flush();
	}

	/// <summary>
	/// Reads one message, or returns null when the stream ends between
	/// messages. A stream that ends part way through one throws
	/// <see cref="EndOfStreamException"/>.
	/// </summary>
	public T? Recv<T>(MessageParser<T> parser) where T : IMessage<T>
	{
		var size = ReadLength();
		if (size is null)
		{
			return default;
		}
		if (size > MaxMessageSize)
		{
			throw new PluginException($"message exceeds the maximum size: {size} bytes");
		}

		var body = new byte[(int)size];
		input.ReadExactly(body);
		try
		{
			return parser.ParseFrom(body);
		}
		catch (InvalidProtocolBufferException e)
		{
			throw new PluginException("unmarshalling: " + e.Message, e);
		}
	}

	// ReadLength reads a varint, or returns null at the end of the stream
	// before its first byte.
	ulong? ReadLength()
	{
		ulong size = 0;
		for (var i = 0; i < MaxVarintLength; i++)
		{
			var b = input.ReadByte();
			if (b < 0)
			{
				if (i == 0)
				{
					return null;
				}
				throw new EndOfStreamException("the stream ended inside a length prefix");
			}
			if (i == MaxVarintLength - 1 && b > 1)
			{
				break;
			}
			size |= (ulong)(b & 0x7f) << (7 * i);
			if (b < 0x80)
			{
				return size;
			}
		}
		throw new PluginException("reading length: the varint overflows 64 bits");
	}
}

/// <summary>A failure of the protocol itself, rather than of the model.</summary>
public sealed class PluginException : Exception
{
	/// <summary>Returns an exception with a message.</summary>
	public PluginException(string message) : base(message) { }

	/// <summary>Returns an exception with a message and its cause.</summary>
	public PluginException(string message, Exception inner) : base(message, inner) { }
}
