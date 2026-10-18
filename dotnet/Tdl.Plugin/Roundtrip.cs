using Tdl.Ir.V1;
using Tdl.Plugin.V1;

namespace Tdl.Plugin;

/// <summary>A top-level item an annotation carries as TDL, with its index among the model's items.</summary>
public readonly record struct CarriedItem(int At, string Source);

public sealed partial class Session
{
	/// <summary>Whether a node carries an argument-less directive for this target.</summary>
	public bool Tagged(IEnumerable<Directive> all, string name) =>
		Directives.For(Target, all).Any(d => d.Name == name);

	/// <summary>Whether this target's block carries an argument-less directive.</summary>
	public bool Bare(string name) =>
		Model.Targets.Any(b => b.Meta?.Name == Target && Tagged(b.Directives, name));

	/// <summary>
	/// Warns that a fact is lost, unless <see cref="Roundtrip"/> is set and an
	/// annotation carries it.
	/// </summary>
	public void Lose(string code, Position? position, string message)
	{
		if (!Roundtrip)
		{
			Lossy(code, position, message);
		}
	}

	/// <summary>
	/// Reports a declaration that is not generated. Under roundtrip it is
	/// carried as TDL instead, so nothing is reported, unless the model is
	/// invalid.
	/// </summary>
	public void Skip(Problem problem)
	{
		if (!Roundtrip || problem is Invalid)
		{
			Warn(problem);
		}
	}

	/// <summary>
	/// Is <see cref="Declares"/> for a target that imports back: what it
	/// cannot generate is a loss, and an alias, expanded where it is used, is
	/// a loss too, with aliased called on its target so the backend can warn
	/// about what the expansion loses.
	/// </summary>
	public bool DeclaresLossy(Decl d, Action<ID> aliased, out Problem? problem)
	{
		var pos = d.Meta?.Position;
		var name = d.Meta?.Name;
		problem = null;

		switch (d.NodeCase)
		{
			case Ir.V1.Decl.NodeOneofCase.Class:
				problem = new Unsupported($"{name} is a class, and classes are not generated yet", pos, LossCodes.Class);
				return false;
			case Ir.V1.Decl.NodeOneofCase.Unit:
				problem = new Unsupported($"{name} is a unit, and units are not generated yet", pos, LossCodes.Unit);
				return false;
			case Ir.V1.Decl.NodeOneofCase.Alias:
				if (d.Alias.Params.Count == 0)
				{
					Lose(LossCodes.Alias, pos, $"alias {name} is expanded where used");
					aliased(d.Alias.Target);
				}
				return false;
			case not (Ir.V1.Decl.NodeOneofCase.Structure or Ir.V1.Decl.NodeOneofCase.Enumeration or Ir.V1.Decl.NodeOneofCase.Newtype):
				// A primitive declares nothing.
				return false;
		}
		if (Params(d).Count > 0)
		{
			problem = new Unsupported($"{name} is parameterized, and generics are not generated yet", pos, LossCodes.Generic);
			return false;
		}
		return true;
	}

	/// <summary>
	/// Warns that an entity or a mixin reads back as a value, and returns its
	/// kind: "entity", "mixin", or "" for anything else.
	/// </summary>
	public string LoseKind(Decl d)
	{
		var name = d.Meta?.Name;
		switch (d.Structure?.Kind)
		{
			case StructKind.Entity:
				Lose(LossCodes.StructKind, d.Meta?.Position, $"{name} is an entity, and reads back as a value");
				return "entity";
			case StructKind.Mixin:
				Lose(LossCodes.StructKind, d.Meta?.Position, $"{name} is a mixin, and reads back as a value");
				return "mixin";
			default:
				return "";
		}
	}

	/// <summary>
	/// Warns about each conformance of a struct or an enum that a target does
	/// not carry, which is all of them but a struct's plain conformance to
	/// Entity, since its kind says that. It reports whether any was lost.
	/// </summary>
	public bool LoseConforms(Decl d, string target)
	{
		var conforms = d.Enumeration?.Conforms ?? d.Structure?.Conforms ?? [];
		var lost = false;
		foreach (var c in conforms)
		{
			var n = c.Class?.Name;
			if (d.Structure is null || n != "Entity" || c.Extern is not null || c.Args.Count > 0)
			{
				Lose(LossCodes.Class, c.Position, $"{d.Meta?.Name} conforms to {n}, which {target} does not carry");
				lost = true;
			}
		}
		return lost;
	}

	/// <summary>
	/// Warns when import would write a name directive and the model has none,
	/// or the reverse: readerWrites is whether the name the node is written
	/// with differs from the convention import reads.
	/// </summary>
	public void CheckRenamed(IEnumerable<Directive> directives, bool readerWrites, string what)
	{
		var d = Find(directives, "name");
		if ((d is not null) == readerWrites)
		{
			return;
		}
		if (d is not null)
		{
			Lose(LossCodes.Name, d.Position, $"{what}'s name directive restates the convention, and does not read back");
			return;
		}
		Lose(LossCodes.Name, null, $"{what} reads back with a name directive");
	}

	/// <summary>
	/// The model's top-level items as TDL, in order. tdl sends them only when
	/// the target block carries the roundtrip directive.
	/// </summary>
	public IReadOnlyList<Item> Items => Analysis.Source?.Items ?? (IReadOnlyList<Item>)[];

	/// <summary>Each import of the model as TDL.</summary>
	public IReadOnlyList<string> Imports => Analysis.Source?.Imports ?? (IReadOnlyList<string>)[];

	/// <summary>A declaration's index among the model's top-level items, or -1.</summary>
	public int IndexOf(string name)
	{
		for (var i = 0; i < Items.Count; i++)
		{
			if (Items[i].Name == name)
			{
				return i;
			}
		}
		return -1;
	}

	/// <summary>
	/// A declaration as TDL, doc comment included, or "" when the model
	/// declares nothing by that name.
	/// </summary>
	public string DeclSource(string name) => IndexOf(name) is var i and >= 0 ? Items[i].Source : "";

	/// <summary>A struct's or an enum's conformance list as TDL: <c>Entity, Auditable</c>.</summary>
	public string Conforms(string name) => IndexOf(name) is var i and >= 0 ? Items[i].Conforms : "";

	/// <summary>
	/// A field as TDL, without its doc comment, and without its deprecation
	/// unless deprecated is set.
	/// </summary>
	public string FieldSource(Field f, bool deprecated)
	{
		var v = View(f);
		var src = v?.Source ?? "";
		var dep = v?.Deprecation ?? "";
		return deprecated && dep != "" ? dep + " " + src : src;
	}

	/// <summary>
	/// Every top-level item that is not an emitted declaration, this target's
	/// block without its roundtrip directive included.
	/// </summary>
	public IReadOnlyList<CarriedItem> Carried(ISet<string> emitted)
	{
		var carried = new List<CarriedItem>();
		for (var i = 0; i < Items.Count; i++)
		{
			var it = Items[i];
			if (it.Name != "" && emitted.Contains(it.Name) || it.Source == "")
			{
				continue;
			}
			carried.Add(new CarriedItem(i, it.Source));
		}
		return carried;
	}
}
