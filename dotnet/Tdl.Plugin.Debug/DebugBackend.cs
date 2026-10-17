using System.Text;
using Google.Protobuf;
using Tdl.Ir.V1;
using Tdl.Plugin.V1;

namespace Tdl.Plugin.Debug;

/// <summary>
/// A port of the Go debug backend, which describes the model it was given.
/// It exercises the SDK and the protocol without generating code, and answers
/// a request with the bytes the Go backend answers it with.
/// </summary>
public sealed class DebugBackend : IBackend
{
	/// <summary>
	/// What this backend is called, in a target block and as
	/// tdl-gen-dotnet_debug on PATH. The Go backend holds the name debug, and
	/// tdl never looks a built-in name up on PATH.
	/// </summary>
	public const string Name = "dotnet_debug";

	/// <inheritdoc/>
	public Description Describe() => new(Name, "0.1.0")
	{
		// Requests share no state.
		Reuse = true,
		Directives =
		[
			// note("...") exists to exercise directive plumbing.
			new DirectiveSpec { Name = "note", MinArgs = 1, MaxArgs = 1, ArgKinds = { LiteralKind.String } },
		],
	};

	/// <summary>Writes one file describing the model.</summary>
	public Response Generate(Request request)
	{
		var s = new Session(request, "debug");
		var model = s.Model;

		var b = new StringBuilder();
		b.Append($"package {model.Package}\n");
		b.Append($"target {request.Target}\n");
		b.Append('\n');

		var own = s.Own();
		b.Append($"declarations: {own.Count} own, {model.Decls.Count - own.Count} from the prelude\n");
		b.Append($"types: {model.Types_.Count}\n");
		b.Append($"externs: {model.Externs.Count}\n");
		b.Append($"instances: {model.Instances.Count}\n");

		foreach (var d in own)
		{
			b.Append($"\n{d.Meta?.Name}\n");
			foreach (var dir in Directives.For(request.Target, d.Directives))
			{
				b.Append($"  directive {dir.Name}\n");
			}
			foreach (var f in Session.Fields(d))
			{
				b.Append($"  field {f.Meta?.Name}: {f.Type?.Name}\n");
			}
		}

		// Each fieldless declaration warns, to exercise the diagnostic path.
		foreach (var d in own)
		{
			if (Session.Fields(d).Count > 0 || d.Enumeration is not null)
			{
				continue;
			}
			s.Warn(new Unsupported($"{d.Meta?.Name} has no fields, so there is nothing to generate for it", d.Meta?.Position));
		}

		return s.Response([new V1.File { Path = "model.txt", Content = ByteString.CopyFromUtf8(b.ToString()) }]);
	}
}
