using Tdl.Ir.V1;
using Tdl.Plugin.V1;
using PluginFile = Tdl.Plugin.V1.File;

namespace Tdl.Plugin;

/// <summary>
/// One request's state: the model, the analysis tdl sent with it, and the
/// diagnostics the backend collects. A backend makes a fresh one per request.
/// </summary>
public sealed partial class Session
{
	readonly Dictionary<Decl, int> decls = new(ReferenceEqualityComparer.Instance);
	readonly Dictionary<object, MemberView> members = new(ReferenceEqualityComparer.Instance);
	readonly Dictionary<Constraint, ConstraintView> constraints = new(ReferenceEqualityComparer.Instance);

	/// <summary>
	/// Starts a session for one request. It throws when the request carries
	/// no analysis for its model, which tdl sends with every request.
	/// </summary>
	public Session(Request request, string lang)
	{
		Model = request.Model ?? new Model();
		Target = request.Target;
		Lang = lang;
		Analysis = request.Analysis
			?? throw new ArgumentException("the request carries no analysis; tdl sends one with every request", nameof(request));
		if (Analysis.Decls.Count != Model.Decls.Count || Analysis.Types_.Count != Model.Types_.Count)
		{
			throw new ArgumentException("the request's analysis does not describe its model", nameof(request));
		}

		for (var i = 0; i < Model.Decls.Count; i++)
		{
			var d = Model.Decls[i];
			var view = Analysis.Decls[i];
			decls[d] = i;
			Index(d.Newtype?.ValueConstraints ?? [], view.Constraints);
			Index(Fields(d), view.Fields);
			var variants = d.Enumeration?.Variants ?? [];
			Index(variants, view.Variants);
			for (var j = 0; j < variants.Count && j < view.Variants.Count; j++)
			{
				Index(variants[j].Fields, view.Variants[j].Fields);
			}
		}
	}

	void Index<T>(IList<T> nodes, IList<MemberView> views) where T : class
	{
		for (var i = 0; i < nodes.Count && i < views.Count; i++)
		{
			members[nodes[i]] = views[i];
			if (nodes[i] is Field f)
			{
				Index(f.Constraints, views[i].Constraints);
			}
		}
	}

	void Index(IList<Constraint> nodes, IList<ConstraintView> views)
	{
		for (var i = 0; i < nodes.Count && i < views.Count; i++)
		{
			constraints[nodes[i]] = views[i];
		}
	}

	/// <summary>The model, with the prelude's declarations merged into it.</summary>
	public Model Model { get; }

	/// <summary>The target block being served.</summary>
	public string Target { get; }

	/// <summary>The target language as a message names it: "C#", "F#".</summary>
	public string Lang { get; }

	/// <summary>What tdl derived from the model for this target.</summary>
	public Analysis Analysis { get; }

	/// <summary>
	/// Makes <see cref="Lose"/> and <see cref="Skip"/> report nothing, since
	/// annotations carry what they would.
	/// </summary>
	public bool Roundtrip { get; set; }

	/// <summary>The diagnostics collected so far.</summary>
	public List<Diagnostic> Diagnostics { get; } = [];

	/// <summary>The response holding files and every diagnostic collected.</summary>
	public Response Response(IEnumerable<PluginFile> files)
	{
		var resp = new Response();
		resp.Files.AddRange(files);
		resp.Diagnostics.AddRange(Diagnostics);
		return resp;
	}

	/// <summary>A struct declaration's fields; an enum's belong to its variants.</summary>
	public static IList<Field> Fields(Decl d) => d.Structure?.Fields ?? (IList<Field>)[];

	/// <summary>The analysis of a declaration in the model, or null for one not in it.</summary>
	public DeclView? View(Decl d) => decls.TryGetValue(d, out var i) ? Analysis.Decls[i] : null;

	/// <summary>The analysis of a field of a declaration in the model, or null.</summary>
	public MemberView? View(Field f) => members.GetValueOrDefault(f);

	/// <summary>The analysis of an enum's variant in the model, or null.</summary>
	public MemberView? View(Variant v) => members.GetValueOrDefault(v);

	/// <summary>
	/// The analysis of a constraint on a newtype or a field in the model, or
	/// null.
	/// </summary>
	public ConstraintView? View(Constraint c) => constraints.GetValueOrDefault(c);

	/// <summary>A constraint as TDL source writes it, as <c>min(0)</c>.</summary>
	public string Source(Constraint c) => View(c)?.Source ?? c.Name;

	/// <summary>The analysis of a type reference, or null when it does not resolve.</summary>
	public TypeView? View(ID? type) =>
		type is { Index: >= 0 } && type.Index < Analysis.Types_.Count ? Analysis.Types_[type.Index] : null;

	/// <summary>The declaration an ID names, or null when it does not resolve.</summary>
	public Decl? DeclAt(ID? id) =>
		id is { Index: >= 0 } && id.Index < Model.Decls.Count ? Model.Decls[id.Index] : null;

	/// <summary>The declarations that did not come from the embedded prelude.</summary>
	public IReadOnlyList<Decl> Own() =>
		Model.Decls.Where((_, i) => Analysis.Decls[i].Own).ToList();

	/// <summary>
	/// The session target's directive with this name carrying at least one
	/// argument, or null.
	/// </summary>
	public Directive? Find(IEnumerable<Directive> all, string name) =>
		Directives.For(Target, all).FirstOrDefault(d => d.Name == name && d.Args.Count > 0);

	/// <summary>The first argument of a directive on a node, as written, or null.</summary>
	public string? Text(IEnumerable<Directive> all, string name) => Find(all, name)?.Args[0].Text;

	/// <summary>
	/// A directive written on the target block itself, returned whole so a
	/// diagnostic about its value has its position.
	/// </summary>
	public Directive? Block(string name) =>
		Model.Targets
			.Where(b => b.Meta?.Name == Target)
			.Select(b => Find(b.Directives, name))
			.FirstOrDefault(d => d is not null);

	/// <summary>
	/// A declaration's name in the target language: a name directive's
	/// argument when there is one, otherwise its words joined by style.
	/// </summary>
	public string DeclName(Decl d, Style style) =>
		Text(d.Directives, "name") ?? style(View(d)?.Words ?? Case.Words(Case.LastSegment(d.Meta?.Name ?? "")));

	/// <summary>A field's name in the target language, by the rule <see cref="DeclName"/> follows.</summary>
	public string FieldName(Field f, Style style) =>
		Text(f.Directives, "name") ?? style(View(f)?.Words ?? Case.Words(f.Meta?.Name ?? ""));

	/// <summary>A variant's name in the target language, by the rule <see cref="DeclName"/> follows.</summary>
	public string VariantName(Variant v, Style style) =>
		Text(v.Directives, "name") ?? style(View(v)?.Words ?? Case.Words(v.Meta?.Name ?? ""));

	/// <summary>
	/// Reports a problem: an <see cref="Unsupported"/> as a warning with its
	/// loss code, and an <see cref="Invalid"/> as an error.
	/// </summary>
	public void Warn(Problem problem)
	{
		var d = new Diagnostic
		{
			Severity = problem is Invalid ? Severity.Error : Severity.Warning,
			Message = problem.Message,
			Position = problem.Position,
		};
		if (problem is Unsupported u)
		{
			d.Code = u.Code;
		}
		Diagnostics.Add(d);
	}

	/// <summary>
	/// Reports a problem that makes the output unusable. The host writes
	/// nothing when a response carries one.
	/// </summary>
	public void Error(Position? position, string message) =>
		Diagnostics.Add(new Diagnostic { Severity = Severity.Error, Message = message, Position = position });

	/// <summary>Warns that a fact is lost, with the loss code that names it.</summary>
	public void Lossy(string code, Position? position, string message) =>
		Diagnostics.Add(new Diagnostic { Severity = Severity.Warning, Message = message, Position = position, Code = code });

	/// <summary>
	/// Warns that a declaration's constraints are not enforced: a newtype's
	/// where block and each field's, including enum variants'. enforced
	/// reports the constraints the target's types enforce themselves, which do
	/// not warn; null enforces none.
	/// </summary>
	public void WarnConstraints(Decl d, Func<Constraint, bool>? enforced = null)
	{
		var name = d.Meta?.Name ?? "";
		if (Unenforced(d.Newtype?.ValueConstraints, enforced) is var n and > 0)
		{
			Lossy(LossCodes.Constraint, d.Meta?.Position,
				$"{name} carries {n} where constraint(s), and validation is not generated yet");
		}

		var fields = Fields(d).Concat((d.Enumeration?.Variants ?? []).SelectMany(v => v.Fields));
		foreach (var f in fields)
		{
			if (Unenforced(f.Constraints, enforced) is var m and > 0)
			{
				Lossy(LossCodes.Constraint, f.Meta?.Position,
					$"{Case.LastSegment(name)}.{f.Meta?.Name} carries {m} constraint(s), and validation is not generated yet");
			}
		}
	}

	static int Unenforced(IEnumerable<Constraint>? cs, Func<Constraint, bool>? enforced) =>
		cs?.Count(c => enforced is null || !enforced(c)) ?? 0;

	/// <summary>
	/// Whether a schema backend writes anything for a declaration, or the
	/// problem that keeps it from generating one: an alias is expanded where
	/// it is used and a primitive names an opaque type, so neither declares
	/// anything, while classes, units, and type parameters are not generated
	/// yet.
	/// </summary>
	public static bool Declares(Decl d, out Problem? problem)
	{
		var pos = d.Meta?.Position;
		var name = d.Meta?.Name;
		problem = d.NodeCase switch
		{
			Ir.V1.Decl.NodeOneofCase.Class => new Unsupported($"{name} is a class, and classes are not generated yet", pos),
			Ir.V1.Decl.NodeOneofCase.Unit => new Unsupported($"{name} is a unit, and units are not generated yet", pos),
			Ir.V1.Decl.NodeOneofCase.Structure or Ir.V1.Decl.NodeOneofCase.Enumeration or Ir.V1.Decl.NodeOneofCase.Newtype when Params(d).Count > 0 =>
				new Unsupported($"{name} is parameterized, and generics are not generated yet", pos),
			_ => null,
		};
		return problem is null && d.NodeCase is Ir.V1.Decl.NodeOneofCase.Structure or Ir.V1.Decl.NodeOneofCase.Enumeration or Ir.V1.Decl.NodeOneofCase.Newtype;
	}

	/// <summary>The type parameters of a declaration.</summary>
	public static IList<Param> Params(Decl d) => d.NodeCase switch
	{
		Ir.V1.Decl.NodeOneofCase.Alias => d.Alias.Params,
		Ir.V1.Decl.NodeOneofCase.Newtype => d.Newtype.Params,
		Ir.V1.Decl.NodeOneofCase.Structure => d.Structure.Params,
		Ir.V1.Decl.NodeOneofCase.Enumeration => d.Enumeration.Params,
		Ir.V1.Decl.NodeOneofCase.Class => d.Class.Params,
		_ => [],
	};

	/// <summary>
	/// A node's documentation, one line per entry, with trailing whitespace
	/// removed and indentation kept.
	/// </summary>
	public static IReadOnlyList<string> Doc(Meta? m) =>
		m?.Doc.Select(line => line.TrimEnd(' ', '\t')).ToList() ?? [];

	/// <summary>
	/// Whether a node is deprecated, with the reason, which may be empty.
	/// </summary>
	public static bool Deprecated(Meta? m, out string reason)
	{
		reason = m?.Deprecated?.Reason ?? "";
		return m?.Deprecated is not null;
	}
}
