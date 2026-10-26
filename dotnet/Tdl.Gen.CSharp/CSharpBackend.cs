using Tdl.Ir.V1;
using Tdl.Plugin;
using Tdl.Plugin.V1;

namespace Tdl.Gen.CSharp;

/// <summary>
/// Generates C# source from a resolved model: a sealed record for each
/// struct, a C# enum for each fieldless enum, an abstract record closed over
/// nested variant records for each enum with fields, and a readonly record
/// struct for each newtype. The mapping is in docs/architecture/csharp-backend.md.
/// </summary>
public sealed class CSharpBackend : IBackend
{
	/// <summary>What this backend is called in a target block, and as tdl-gen-csharp.</summary>
	public const string Name = "csharp";

	/// <inheritdoc/>
	public Description Describe()
	{
		Placement[] members = [.. Directives.OnMembers];
		return new(Name, "0.1.0")
		{
			// Each request stands alone.
			Reuse = true,
			Directives =
			[
				// The namespace, on the target block.
				Spec("namespace", 1, 1, [LiteralKind.String], [Placement.Block]),
				// One file for every declaration, on the target block.
				Spec("file", 1, 1, [LiteralKind.String], [Placement.Block]),
				// The C# identifier for a declaration, field, or variant.
				Spec("name", 1, 1, [LiteralKind.String], members),
				// The namespace and type a declaration maps to, declared
				// elsewhere.
				Spec("foreign", 2, 2, [LiteralKind.String, LiteralKind.String], [Placement.Decl]),
				// The fields identifying an entity. ArgKinds constrains by
				// position, so the key checks that each is a name.
				Spec("key", 1, -1, [], [Placement.Decl]),
				// An attribute, by its namespace and name, on a declaration or
				// on every declaration from the target block.
				Spec("attribute", 2, 2, [LiteralKind.String, LiteralKind.String], [Placement.Block, Placement.Decl], repeatable: true),
			],
		};
	}

	static DirectiveSpec Spec(string name, int min, int max, LiteralKind[] kinds, Placement[] placements, bool repeatable = false)
	{
		var spec = new DirectiveSpec { Name = name, MinArgs = min, MaxArgs = max, Repeatable = repeatable };
		spec.ArgKinds.AddRange(kinds);
		spec.Placements.AddRange(placements);
		return spec;
	}

	/// <summary>
	/// Returns one file per declaration the model owns, or one file holding
	/// them all when the target block names it.
	/// </summary>
	public Response Generate(Request request) => new Generator(new Session(request, "C#")).Generate();
}
