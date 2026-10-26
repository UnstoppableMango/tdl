using Tdl.Ir.V1;

namespace Tdl.Plugin;

/// <summary>Something a backend reports about the model, at a position.</summary>
public abstract record Problem(string Message, Position? Position);

/// <summary>
/// A shape the backend cannot express. It reaches the user as a warning,
/// carrying a loss code from <see cref="LossCodes"/> when its omission is one
/// the user may choose to allow.
/// </summary>
public sealed record Unsupported(string Message, Position? Position, string Code = "") : Problem(Message, Position);

/// <summary>
/// A directive the user got wrong, such as two fields pinned to one number.
/// It reaches the user as an error, so the host writes nothing.
/// </summary>
public sealed record Invalid(string Message, Position? Position) : Problem(Message, Position);

/// <summary>
/// Loss codes name a fact a conversion cannot carry, in either direction. A
/// warning carrying one can be silenced in tdl.toml or with --allow-lossy.
/// docs/architecture/reverse.md describes each.
/// </summary>
public static class LossCodes
{
#pragma warning disable CS1591 // Each is described in reverse.md.
	public const string Primitive = "lossy.primitive";
	public const string Newtype = "lossy.newtype";
	public const string Collection = "lossy.collection";
	public const string Optional = "lossy.optional";
	public const string StructKind = "lossy.struct-kind";
	public const string Include = "lossy.include";
	public const string Alias = "lossy.alias";
	public const string Constraint = "lossy.constraint";
	public const string Owned = "lossy.owned";
	public const string Default = "lossy.default";
	public const string Key = "lossy.key";
	public const string Name = "lossy.name";
	public const string Number = "lossy.number";
	public const string Order = "lossy.order";
	public const string Doc = "lossy.doc";
	public const string Generic = "lossy.generic";
	public const string Class = "lossy.class";
	public const string Unit = "lossy.unit";
	public const string Unsupported = "lossy.unsupported";
#pragma warning restore CS1591

	/// <summary>Every loss code, in the order the design lists them.</summary>
	public static IReadOnlyList<string> All { get; } =
	[
		Primitive, Newtype, Collection, Optional, StructKind,
		Include, Alias, Constraint, Owned, Default, Key,
		Name, Number, Order, Doc, Generic, Class,
		Unit, Unsupported,
	];
}
