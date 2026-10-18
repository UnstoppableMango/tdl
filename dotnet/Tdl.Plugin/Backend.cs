using Tdl.Ir.V1;
using Tdl.Plugin.V1;

namespace Tdl.Plugin;

/// <summary>A backend turns a resolved model into files.</summary>
public interface IBackend
{
	/// <summary>
	/// Reports what this backend is and what directives it understands. tdl
	/// checks a target block against them before generating.
	/// </summary>
	Description Describe();

	/// <summary>
	/// Returns the files for one request. It throws only when it cannot
	/// produce a response at all; a problem with the model belongs in the
	/// response's diagnostics.
	/// </summary>
	Response Generate(Request request);
}

/// <summary>
/// A backend that also reads its target language back into a model, for
/// <c>tdl import</c>. Its description sets <see cref="Description.Reverse"/>.
/// </summary>
public interface IImporter : IBackend
{
	/// <summary>
	/// Returns the model the request's files describe, or TDL source the host
	/// lowers. It throws only when it cannot produce a response at all.
	/// </summary>
	ImportResponse Import(ImportRequest request);
}

/// <summary>What a backend says about itself.</summary>
public sealed record Description(string Name, string Version)
{
	/// <summary>The directives the backend understands.</summary>
	public IReadOnlyList<DirectiveSpec> Directives { get; init; } = [];

	/// <summary>Says the backend can serve more than one request on a connection.</summary>
	public bool Reuse { get; init; }

	/// <summary>Says the backend is an <see cref="IImporter"/>.</summary>
	public bool Reverse { get; init; }

	/// <summary>
	/// How the backend numbers fields and variants, or null for a backend
	/// that does not.
	/// </summary>
	public Numbering? Numbering { get; init; }

	/// <summary>The handshake reply carrying this description.</summary>
	public HandshakeReply Reply()
	{
		var reply = new HandshakeReply
		{
			Accepted = true,
			Name = Name,
			Version = Version,
			Features = new Features { Reuse = Reuse, Reverse = Reverse },
			Numbering = Numbering,
		};
		reply.Directives.AddRange(Directives);
		return reply;
	}
}

/// <summary>Builds the directive specs a description lists.</summary>
public static class Directives
{
	/// <summary>
	/// The directives in all that belong to target. A request from tdl
	/// carries only its target's directives, so this matters for a model
	/// built some other way.
	/// </summary>
	public static IEnumerable<Directive> For(string target, IEnumerable<Directive> all) =>
		all.Where(d => d.Target == target);

	/// <summary>Where a name for a declaration, a field, or a variant goes.</summary>
	public static Placement[] OnMembers { get; } = [Placement.Decl, Placement.Field, Placement.Variant];
}
