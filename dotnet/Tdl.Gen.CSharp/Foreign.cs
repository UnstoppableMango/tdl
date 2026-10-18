using Tdl.DotNet;
using Tdl.Ir.V1;
using Tdl.Plugin;

namespace Tdl.Gen.CSharp;

// A `foreign` directive maps a declaration, a primitive included, to a type
// another assembly or namespace declares, which the generated code names
// instead of declaring. An extern without one is the type its own package's
// csharp target block generates.
sealed partial class Generator
{
	// foreign and externs map a declaration and an extern to the type
	// another assembly or namespace declares for it, written qualified.
	readonly Dictionary<Decl, string> foreign = new(ReferenceEqualityComparer.Instance);
	readonly Dictionary<Extern, string> externs = new(ReferenceEqualityComparer.Instance);

	// PlanForeign reads every mapping before rendering: each declaration's,
	// the prelude's included, and each extern's, from a mapping or else from
	// its dependency's csharp target block.
	void PlanForeign()
	{
		for (var i = 0; i < s.Model.Decls.Count; i++)
		{
			var d = s.Model.Decls[i];
			if (Mapping(d.Directives, d.Meta?.Name ?? "") is not { } t)
			{
				continue;
			}
			foreign[d] = t;
			if (!s.Analysis.Decls[i].Own)
			{
				continue;
			}
			WarnForeignConstraints(d);
			if (s.Find(d.Directives, "key") is { } key)
			{
				s.Lossy(LossCodes.Key, key.Position,
					$"{d.Meta?.Name} is a foreign type, so its key is not generated: the key method needs the type's properties");
			}
		}

		foreach (var e in s.Model.Externs)
		{
			if ((Mapping(e.Directives, $"{e.Package}.{e.Name}") ?? Dependency(e)) is { } t)
			{
				externs[e] = t;
			}
		}
	}

	// Mapping reads the foreign directive among dirs as the type it names,
	// warning when it is unusable.
	string? Mapping(IEnumerable<Directive> dirs, string of)
	{
		if (s.Find(dirs, "foreign") is not { } dir)
		{
			return null;
		}
		if (!Net.Parse(Texts(dir), out var q, out var problem))
		{
			s.Warn(new Plugin.Unsupported($"the foreign mapping of {of} {problem}", dir.Position));
			return null;
		}
		return Qualify(q);
	}

	// Dependency is the type an extern's own package generates, when that
	// package has a csharp target block: in the block's namespace, or else
	// the one its package derives, under the extern's `name` or its own name
	// with the first letter upper-cased.
	string? Dependency(Extern e)
	{
		var dep = s.Model.Imports.FirstOrDefault(i => i.Package == e.Package);
		if (dep is null || !Directives.For(s.Target, dep.Directives).Any())
		{
			return null;
		}
		var ns = s.Text(dep.Directives, "namespace") ?? Net.Namespace(dep.Package);
		var name = s.Text(e.Directives, "name") ?? Net.UpperFirst(e.Name);
		if (!Net.Parse([ns, name], out var q, out var problem))
		{
			s.Warn(new Plugin.Unsupported($"{e.Package}.{e.Name} is generated as a type that {problem}", e.Position));
			return null;
		}
		return Qualify(q);
	}

	// WarnForeignConstraints warns at each constraint on a foreign
	// declaration, since the backend does not know what the type holds.
	void WarnForeignConstraints(Decl d)
	{
		var fields = Session.Fields(d).Concat((d.Enumeration?.Variants ?? []).SelectMany(v => v.Fields));
		var cs = (d.Newtype?.ValueConstraints ?? []).Concat(fields.SelectMany(f => f.Constraints));
		foreach (var c in cs)
		{
			s.Lossy(LossCodes.Constraint, c.Position,
				$"{s.Source(c)} is not checked: {d.Meta?.Name} is a foreign type, whose values another assembly decides");
		}
	}

	// ForeignPrim is the type a primitive is mapped to, if any.
	string? ForeignPrim(ID id) =>
		s.DeclAt(s.Model.Types_[id.Index].Ctor) is { } d && foreign.TryGetValue(d, out var t) ? t : null;

	// AttributesOf renders every attribute directive among dirs as the name
	// of the attribute, warning at one that is unusable.
	List<string> AttributesOf(IEnumerable<Directive> dirs)
	{
		var output = new List<string>();
		foreach (var dir in Directives.For(s.Target, dirs).Where(d => d.Name == "attribute"))
		{
			if (!Net.Parse(Texts(dir), out var q, out var problem))
			{
				s.Warn(new Plugin.Unsupported($"attribute {problem}", dir.Position));
				continue;
			}
			output.Add(Qualify(q));
		}
		return output;
	}

	// BlockDirectives are the directives written on the target block itself.
	IEnumerable<Directive> BlockDirectives() =>
		s.Model.Targets.Where(b => b.Meta?.Name == s.Target).SelectMany(b => b.Directives);

	// Qualify writes a type another namespace declares from the global
	// namespace, escaping each segment C# reserves.
	static string Qualify(Qualified q) =>
		"global::" + string.Join('.', q.Namespace.Split('.').Select(Escape)) + "." + Escape(q.Name);

	static string[] Texts(Directive dir) => [.. dir.Args.Select(a => a.Text)];
}
