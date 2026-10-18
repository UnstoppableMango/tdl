using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;
using Tdl.Ir.V1;
using Tdl.Plugin;
using static Microsoft.CodeAnalysis.CSharp.SyntaxFactory;

namespace Tdl.Gen.CSharp;

sealed partial class Generator
{
	// Key renders an entity's identity, which a `key` directive names: a Key
	// method returning the one property it names, or one building a key
	// record struct holding each, which after declares beside the entity. On
	// a problem the caller warns and emits the entity without its key.
	(MethodDeclarationSyntax Method, MemberDeclarationSyntax? After) Key(Decl d, string typ, List<Property> props, Directive dir)
	{
		var pos = dir.Position;
		Position? At(Literal arg) => arg.Position ?? pos;
		var name = d.Meta?.Name;
		if (d.Structure.Kind != StructKind.Entity)
		{
			throw Unsupported(pos, $"{name} has a key, and only an entity is identified by one", LossCodes.Key);
		}
		if (props.Any(p => p.Name == "Key"))
		{
			throw Unsupported(pos, $"{name} has a property Key, which the generated key method would be named", LossCodes.Key);
		}

		var keyed = new List<Property>();
		foreach (var arg in dir.Args)
		{
			if (arg.Kind != LiteralKind.Name)
			{
				throw Unsupported(At(arg), $"a key names fields, and {Literals.Quote(arg.Text)} is {Literals.KindName(arg.Kind)}", LossCodes.Key);
			}
			var p = props.Find(p => p.Field.Meta?.Name == arg.Text)
				?? throw Unsupported(At(arg), $"{name} has no field {arg.Text} to key on", LossCodes.Key);
			if (keyed.Any(k => ReferenceEquals(k.Field, p.Field)))
			{
				throw Unsupported(At(arg), $"the key of {name} names {arg.Text} twice", LossCodes.Key);
			}
			if (p.Type.Cmp is not null)
			{
				throw Unsupported(At(arg),
					$"{name}.{arg.Text} cannot be part of a key in C#, which compares a collection or bytes by reference", LossCodes.Key);
			}
			keyed.Add(p);
		}
		if (keyed.Count == 0)
		{
			throw Unsupported(pos, $"the key of {name} names no field", LossCodes.Key);
		}

		if (keyed.Count == 1)
		{
			return (KeyMethod(keyed[0].Type.Text, IdentifierName(Syntax.Name(keyed[0].Name))), null);
		}

		var record = Bare(typ) + "Key";
		if (owners.TryGetValue(record, out var other))
		{
			throw Unsupported(pos, $"the key record {record} would collide with {other}", LossCodes.Key);
		}
		if (keyed.Any(p => Bare(p.Name) == record))
		{
			throw Unsupported(pos, $"the key record {record} cannot hold a property of its own name", LossCodes.Key);
		}
		var after = Syntax.RecordStruct(record, keyed.Select(p => (p.Type.Text, p.Name)))
			.WithSemicolonToken(Token(SyntaxKind.SemicolonToken))
			.WithLeadingTrivia(ParseLeadingTrivia($"/// <summary>The fields identifying a <see cref=\"{typ}\"/>.</summary>\n"));
		var args = ArgumentList(SeparatedList(keyed.Select(p => Argument(IdentifierName(Syntax.Name(p.Name))))));
		return (KeyMethod(record, ImplicitObjectCreationExpression(args, null)), after);
	}

	static MethodDeclarationSyntax KeyMethod(string type, ExpressionSyntax body) =>
		MethodDeclaration(ParseTypeName(type), "Key")
			.WithModifiers(Syntax.Modifiers(SyntaxKind.PublicKeyword))
			.WithExpressionBody(ArrowExpressionClause(body))
			.WithSemicolonToken(Token(SyntaxKind.SemicolonToken));
}
