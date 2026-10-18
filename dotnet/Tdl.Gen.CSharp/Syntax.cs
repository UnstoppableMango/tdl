using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;
using static Microsoft.CodeAnalysis.CSharp.SyntaxFactory;

namespace Tdl.Gen.CSharp;

/// <summary>
/// Builds and lays out the syntax the generator writes. Roslyn's
/// NormalizeWhitespace indents and places braces; Layout then decides blank
/// lines and wraps a generated Equals, which it leaves to a single line.
/// </summary>
static class Syntax
{
	const string Indent = "    ";

	/// <summary>A name as written, a keyword with its <c>@</c>.</summary>
	public static SyntaxToken Name(string written) =>
		written.StartsWith('@') ? VerbatimIdentifier(default, written[1..], written[1..], default) : Identifier(written);

	public static SyntaxTokenList Modifiers(params SyntaxKind[] kinds) => TokenList(kinds.Select(Token));

	public static AttributeListSyntax Attribute(string name, ExpressionSyntax? arg = null)
	{
		var attr = SyntaxFactory.Attribute(ParseName(name));
		if (arg is not null)
		{
			attr = attr.WithArgumentList(AttributeArgumentList(SingletonSeparatedList(AttributeArgument(arg))));
		}
		return AttributeList(SingletonSeparatedList(attr));
	}

	public static LiteralExpressionSyntax String(string s) =>
		LiteralExpression(SyntaxKind.StringLiteralExpression, Literal(s));

	/// <summary>
	/// Documentation as an XML doc comment, on one line when it is one line.
	/// The lines are escaped here.
	/// </summary>
	public static SyntaxTriviaList Doc(IReadOnlyList<string> lines) => lines.Count switch
	{
		0 => default,
		1 => ParseLeadingTrivia($"/// <summary>{Xml(lines[0])}</summary>\n"),
		_ => ParseLeadingTrivia("/// <summary>\n" + string.Concat(lines.Select(l => ("/// " + Xml(l)).TrimEnd(' ') + "\n")) + "/// </summary>\n"),
	};

	static string Xml(string s) => s.Replace("&", "&amp;").Replace("<", "&lt;").Replace(">", "&gt;");

	/// <summary>
	/// A property with get and init accessors, required unless it has an
	/// initializer.
	/// </summary>
	public static PropertyDeclarationSyntax Property(string type, string name, bool required, ExpressionSyntax? init)
	{
		var p = PropertyDeclaration(ParseTypeName(type), Name(name))
			.WithModifiers(required ? Modifiers(SyntaxKind.PublicKeyword, SyntaxKind.RequiredKeyword) : Modifiers(SyntaxKind.PublicKeyword))
			.WithAccessorList(AccessorList(List(
			[
				AccessorDeclaration(SyntaxKind.GetAccessorDeclaration).WithSemicolonToken(Token(SyntaxKind.SemicolonToken)),
				AccessorDeclaration(SyntaxKind.InitAccessorDeclaration).WithSemicolonToken(Token(SyntaxKind.SemicolonToken)),
			])));
		return init is null ? p : p.WithInitializer(EqualsValueClause(init)).WithSemicolonToken(Token(SyntaxKind.SemicolonToken));
	}

	/// <summary>A record, empty when it has no members.</summary>
	public static RecordDeclarationSyntax Record(SyntaxTokenList modifiers, string name, string? baseType, IReadOnlyList<MemberDeclarationSyntax> members)
	{
		var r = RecordDeclaration(Token(SyntaxKind.RecordKeyword), Name(name)).WithModifiers(modifiers);
		if (baseType is not null)
		{
			r = r.AddBaseListTypes(SimpleBaseType(ParseTypeName(baseType)));
		}
		return members.Count == 0
			? r.WithSemicolonToken(Token(SyntaxKind.SemicolonToken))
			: r.WithOpenBraceToken(Token(SyntaxKind.OpenBraceToken)).WithMembers(List(members)).WithCloseBraceToken(Token(SyntaxKind.CloseBraceToken));
	}

	/// <summary>A readonly record struct with a primary constructor.</summary>
	public static RecordDeclarationSyntax RecordStruct(string name, IEnumerable<(string Type, string Name)> parameters) =>
		RecordDeclaration(SyntaxKind.RecordStructDeclaration, Token(SyntaxKind.RecordKeyword), Name(name))
			.WithClassOrStructKeyword(Token(SyntaxKind.StructKeyword))
			.WithModifiers(Modifiers(SyntaxKind.PublicKeyword, SyntaxKind.ReadOnlyKeyword, SyntaxKind.PartialKeyword))
			.WithParameterList(ParameterList(SeparatedList(parameters.Select(p => Parameter(Name(p.Name)).WithType(ParseTypeName(p.Type))))));

	/// <summary>
	/// The text of a top-level member and what follows it, each laid out and
	/// separated by a blank line.
	/// </summary>
	public static string Text(IEnumerable<MemberDeclarationSyntax> members) =>
		string.Join("\n", members.Select(m => Layout(m).ToFullString().TrimEnd('\n') + "\n"));

	static SyntaxNode Layout(MemberDeclarationSyntax member)
	{
		var node = member.NormalizeWhitespace(Indent, "\n");
		node = node.ReplaceTokens(Tokens(node), Retrivia);
		return node.ReplaceNodes(
			node.DescendantNodesAndSelf().OfType<TypeDeclarationSyntax>().SelectMany(t => t.Members.Skip(1)),
			(original, current) =>
			{
				var members = ((TypeDeclarationSyntax)original.Parent!).Members;
				var blank = Blank(members[members.IndexOf(original) - 1], original);
				var trivia = current.GetLeadingTrivia().SkipWhile(t => t.IsKind(SyntaxKind.EndOfLineTrivia));
				return current.WithLeadingTrivia(blank ? trivia.Prepend(EndOfLine("\n")) : trivia);
			});
	}

	// Blank decides whether a blank line separates two members of a type:
	// comparer fields and undocumented properties run together, and
	// everything else stands apart.
	static bool Blank(MemberDeclarationSyntax a, MemberDeclarationSyntax b) =>
		(a, b) switch
		{
			(FieldDeclarationSyntax, FieldDeclarationSyntax) => false,
			(PropertyDeclarationSyntax, PropertyDeclarationSyntax) => Documented(a) || Documented(b),
			_ => true,
		};

	static bool Documented(SyntaxNode n) =>
		n.GetLeadingTrivia().Any(t => t.IsKind(SyntaxKind.SingleLineDocumentationCommentTrivia));

	// Tokens are those Retrivia moves: an empty constructor's braces, and the
	// arrow and operators of a generated Equals with what precedes each.
	static HashSet<SyntaxToken> Tokens(SyntaxNode node)
	{
		var tokens = new HashSet<SyntaxToken>();
		foreach (var ctor in node.DescendantNodes().OfType<ConstructorDeclarationSyntax>())
		{
			if (ctor.Body is { Statements.Count: 0 } body)
			{
				tokens.UnionWith([ctor.ParameterList.CloseParenToken, body.OpenBraceToken, body.CloseBraceToken]);
			}
		}
		foreach (var attr in node.DescendantNodes(descendIntoTrivia: true).OfType<XmlAttributeSyntax>())
		{
			tokens.UnionWith([attr.Name.LocalName, attr.EqualsToken]);
		}
		foreach (var token in node.DescendantTokens().Where(t => t.TrailingTrivia.Count(IsLineEnd) > 1))
		{
			tokens.Add(token);
		}
		foreach (var eq in node.DescendantNodes().OfType<MethodDeclarationSyntax>().Where(Wrapped))
		{
			tokens.Add(eq.ExpressionBody!.ArrowToken);
			foreach (var op in Operators(eq))
			{
				tokens.UnionWith([op, op.GetPreviousToken()]);
			}
		}
		return tokens;
	}

	static bool IsLineEnd(SyntaxTrivia t) => t.IsKind(SyntaxKind.EndOfLineTrivia);

	static bool Wrapped(MethodDeclarationSyntax m) =>
		m.Identifier.Text == "Equals" && m.ExpressionBody?.Expression is BinaryExpressionSyntax;

	static IEnumerable<SyntaxToken> Operators(MethodDeclarationSyntax eq) =>
		eq.ExpressionBody!.Expression.DescendantNodesAndSelf()
			.OfType<BinaryExpressionSyntax>()
			.Where(b => b.IsKind(SyntaxKind.LogicalOrExpression) || b.IsKind(SyntaxKind.LogicalAndExpression))
			.Select(b => b.OperatorToken);

	// Retrivia leaves one line ending after a token, for Layout to add the
	// blank lines; writes an empty constructor as `X() { }`, an XML doc
	// attribute as name="value", and an Equals as one comparison per line.
	static SyntaxToken Retrivia(SyntaxToken original, SyntaxToken token)
	{
		if (token.TrailingTrivia.Count(IsLineEnd) > 1)
		{
			token = token.WithTrailingTrivia(token.TrailingTrivia.TakeWhile(t => !IsLineEnd(t)).Append(EndOfLine("\n")));
		}

		var parent = original.Parent;
		if (parent is XmlNameSyntax { Parent: XmlAttributeSyntax } || original.IsKind(SyntaxKind.EqualsToken) && parent is XmlAttributeSyntax)
		{
			return token.WithTrailingTrivia().WithLeadingTrivia(original.IsKind(SyntaxKind.EqualsToken) ? default : token.LeadingTrivia);
		}
		if (parent?.Parent is ConstructorDeclarationSyntax || parent is ParameterListSyntax { Parent: ConstructorDeclarationSyntax })
		{
			return original.Kind() switch
			{
				SyntaxKind.CloseParenToken => token.WithTrailingTrivia(Space),
				SyntaxKind.OpenBraceToken => token.WithLeadingTrivia().WithTrailingTrivia(Space),
				SyntaxKind.CloseBraceToken => token.WithLeadingTrivia(),
				_ => token,
			};
		}
		if (parent?.AncestorsAndSelf().OfType<MethodDeclarationSyntax>().FirstOrDefault() is not { } eq || !Wrapped(eq))
		{
			return token;
		}

		var depth = eq.Ancestors().OfType<TypeDeclarationSyntax>().Count() + 1;
		var line = TriviaList(EndOfLine("\n"), Whitespace(string.Concat(Enumerable.Repeat(Indent, depth))));
		var operators = Operators(eq).ToHashSet();
		if (original == eq.ExpressionBody!.ArrowToken)
		{
			return token.WithTrailingTrivia(line);
		}
		if (operators.Contains(original))
		{
			return token.WithLeadingTrivia(line);
		}
		// The token before an operator ends its line.
		return operators.Contains(original.GetNextToken()) ? token.WithTrailingTrivia() : token;
	}
}
