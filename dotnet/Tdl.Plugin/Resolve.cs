using Tdl.Ir.V1;
using Tdl.Plugin.V1;

namespace Tdl.Plugin;

/// <summary>What a resolved type reference is.</summary>
public enum Shape
{
	/// <summary>A primitive, named by <see cref="Ref.Name"/>.</summary>
	Prim,

	/// <summary>A value that may be absent: <c>T?</c>.</summary>
	Option,

	/// <summary>A value that may be null: <c>T | null</c>.</summary>
	Nullable,

	/// <summary>A list of <see cref="Ref.Elem"/>.</summary>
	List,

	/// <summary>A set of <see cref="Ref.Elem"/>.</summary>
	Set,

	/// <summary>A map from <see cref="Ref.Key"/> to <see cref="Ref.Elem"/>.</summary>
	Map,

	/// <summary>A declaration another package holds, <see cref="Ref.Extern"/>.</summary>
	Extern,

	/// <summary>A declaration in the model, <see cref="Ref.Decl"/>.</summary>
	Named,
}

/// <summary>
/// A type reference resolved to the prelude's shape it has or the
/// declaration it names, with aliases expanded.
/// </summary>
public sealed record Ref(Shape Shape, ID Id, Position? Position)
{
	/// <summary>A primitive's name, or an extern's.</summary>
	public string Name { get; init; } = "";

	/// <summary>The element of an option, a nullable, a collection, or the value of a map.</summary>
	public Ref? Elem { get; init; }

	/// <summary>A map's key.</summary>
	public Ref? Key { get; init; }

	/// <summary>The declaration a named reference names.</summary>
	public Decl? Decl { get; init; }

	/// <summary>The extern an extern reference names.</summary>
	public Extern? Extern { get; init; }
}

/// <summary>Carries a problem out of code that cannot go on, to the code that reports it.</summary>
public sealed class ProblemException(Problem problem) : Exception(problem.Message)
{
	/// <summary>What went wrong, and where.</summary>
	public Problem Problem { get; } = problem;
}

public sealed partial class Session
{
	/// <summary>
	/// Makes <see cref="Resolve"/> resolve a reference to another package's
	/// declaration, for a backend that maps one; otherwise it refuses it.
	/// </summary>
	public bool Externs { get; set; }

	/// <summary>
	/// The type a reference names, as the analysis resolved it. It throws a
	/// <see cref="ProblemException"/> carrying the refusal, worded for
	/// <see cref="Lang"/>, when the type has no shape this SDK names.
	/// </summary>
	public Ref Resolve(ID? id)
	{
		var v = View(id) ?? throw Refused(new Unsupported($"type {id?.Name} did not resolve", null));
		if (v.Refusal is { Reason: not Refusal.Types.Reason.MissingArg } r)
		{
			throw Refused(Refuse(r));
		}

		var at = v.Id is { } viewed && viewed.Index != id!.Index ? viewed : id!;
		var pos = Model.Types_[at.Index].Position;
		var t = Model.Types_[at.Index];

		// An element or a key, which a missing type argument leaves unset.
		Ref Part(ID? part) => part is null ? throw Refused(Refuse(v.Refusal)) : Resolve(part);

		switch (v.Form)
		{
			case Form.Extern:
				var i = v.Extern?.Index ?? -1;
				if (!Externs || i < 0 || i >= Model.Externs.Count)
				{
					throw Refused(new Unsupported($"{t.Extern?.Name} is declared in another package, and foreign types are not generated yet", pos));
				}
				return new Ref(Shape.Extern, at, pos) { Name = t.Extern?.Name ?? "", Extern = Model.Externs[i] };
			case Form.Primitive:
				return new Ref(Shape.Prim, at, pos) { Name = v.Name };
			case Form.List or Form.Set:
				return new Ref(v.Form == Form.Set ? Shape.Set : Shape.List, at, pos) { Elem = Part(v.Elem) };
			case Form.Map:
				var key = Part(v.Key);
				return new Ref(Shape.Map, at, pos) { Key = key, Elem = Part(v.Elem) };
			case Form.Option or Form.Nullable:
				return new Ref(v.Form == Form.Nullable ? Shape.Nullable : Shape.Option, at, pos) { Elem = Part(v.Elem) };
			default:
				return new Ref(Shape.Named, at, pos) { Decl = DeclAt(v.Decl) };
		}
	}

	static ProblemException Refused(Problem p) => new(p);

	// Refuse words a refusal from the analysis as this target's warning.
	Unsupported Refuse(Refusal? r)
	{
		var pos = r?.Position;
		var subject = r?.Subject;
		return new Unsupported(r?.Reason switch
		{
			Refusal.Types.Reason.Param => $"type parameter {subject}: generics are not generated yet",
			Refusal.Types.Reason.Unit => $"a unit-typed field has no {Lang} type yet",
			Refusal.Types.Reason.Class => $"{subject} is a class, and a class is not a {Lang} type",
			Refusal.Types.Reason.UnitDecl => $"{subject} is a unit, and units are not generated yet",
			Refusal.Types.Reason.TypeArgs => $"{subject} is applied to type arguments, and generics are not generated yet",
			Refusal.Types.Reason.MissingArg => $"{subject} is missing a type argument",
			_ => $"type {subject} did not resolve",
		}, pos);
	}
}
