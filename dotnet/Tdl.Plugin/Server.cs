using Tdl.Plugin.V1;

namespace Tdl.Plugin;

/// <summary>Serves a backend as a plugin.</summary>
public static class Server
{
	/// <summary>
	/// Runs a backend over standard input and output, reading requests until
	/// the stream ends, and returns the process's exit code. A plugin's Main
	/// is this and nothing else.
	/// </summary>
	public static int Run(IBackend backend)
	{
		try
		{
			using var stdin = Console.OpenStandardInput();
			using var stdout = Console.OpenStandardOutput();
			Serve(backend, new Conn(stdin, stdout));
			return 0;
		}
		catch (Exception e)
		{
			// A backend that throws cannot answer; tdl reports what it wrote.
			Console.Error.WriteLine("plugin: " + e.Message);
			return 1;
		}
	}

	/// <summary>
	/// Serves one connection: the handshake, then requests until the stream
	/// ends. A refused handshake is answered and then thrown.
	/// </summary>
	public static void Serve(IBackend backend, Conn conn)
	{
		var hello = conn.Recv(Handshake.Parser) ?? throw new EndOfStreamException("the stream ended before the handshake");

		var desc = backend.Describe();
		var refusal = Incompatible(hello);
		if (refusal is null && hello.Mode == Mode.Import && (backend is not IImporter || !desc.Reverse))
		{
			refusal = desc.Name + " does not import; it generates only";
		}
		if (refusal is not null)
		{
			conn.Send(new HandshakeReply { Accepted = false, Refusal = refusal });
			throw new PluginException(refusal);
		}

		conn.Send(desc.Reply());

		if (hello.Mode == Mode.Import)
		{
			var importer = (IImporter)backend;
			while (conn.Recv(ImportRequest.Parser) is { } req)
			{
				conn.Send(importer.Import(req));
			}
			return;
		}

		// Several requests arrive only in a watch session with a backend that
		// declared reuse.
		while (conn.Recv(Request.Parser) is { } req)
		{
			conn.Send(backend.Generate(req));
		}
	}

	// Incompatible says what a plugin needed when it cannot read what the
	// host is about to send, or returns null when it can.
	static string? Incompatible(Handshake h)
	{
		if (h.FramingVersion != Conn.FramingVersion)
		{
			return $"framing version {h.FramingVersion} is not supported; this plugin speaks {Conn.FramingVersion}";
		}
		if (h.IrVersion != Conn.IrVersion)
		{
			return $"model version \"{h.IrVersion}\" is not supported; this plugin reads \"{Conn.IrVersion}\"";
		}
		return null;
	}
}
