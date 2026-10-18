using Tdl.Ir.V1;

namespace Tdl.Plugin;

public sealed partial class Session
{
	/// <summary>
	/// The declarations d names through its fields, its variants' fields, a
	/// newtype's base, and their type arguments, with aliases expanded. A
	/// declaration naming itself is included.
	/// </summary>
	public IReadOnlyList<Decl> References(Decl d) => Decls(View(d)?.References ?? []);

	/// <summary>
	/// The declarations the type references name, walked as
	/// <see cref="References"/> walks a declaration's, each once.
	/// </summary>
	public IReadOnlyList<Decl> TypeReferences(params ID[] types)
	{
		var seen = new HashSet<int>();
		var refs = types
			.Select(View)
			.SelectMany(v => v?.References ?? [])
			.Where(r => seen.Add(r.Index));
		return Decls(refs);
	}

	IReadOnlyList<Decl> Decls(IEnumerable<ID> ids) =>
		ids.Select(DeclAt).OfType<Decl>().ToList();

	/// <summary>
	/// Extends skipped, transitively, with every declaration in decls that
	/// names a skipped one, and warns about each it adds, so the output never
	/// refers to something it does not declare.
	/// </summary>
	public void Cascade(IEnumerable<Decl> decls, ISet<Decl> skipped)
	{
		var all = decls.ToList();
		for (var changed = true; changed;)
		{
			changed = false;
			foreach (var d in all)
			{
				// An alias emits nothing; References already expands it.
				if (skipped.Contains(d) || d.Alias is not null)
				{
					continue;
				}
				var named = References(d).FirstOrDefault(skipped.Contains);
				if (named is null)
				{
					continue;
				}
				skipped.Add(d);
				changed = true;
				Warn(new Unsupported($"{d.Meta?.Name} names {named.Meta?.Name}, which is not generated", d.Meta?.Position));
			}
		}
	}
}
