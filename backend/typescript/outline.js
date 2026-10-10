// outline reads TypeScript files with the compiler API and writes, as JSON,
// the outline backend/typescript/reverse.go reads: each statement's kind,
// name, JSDoc comments, members, and types. It is given the TypeScript
// package's directory as its argument and a JSON array of {path, content}
// on stdin.
"use strict";

const ts = require(process.argv[process.argv.length - 1]);

function main() {
	const input = JSON.parse(require("node:fs").readFileSync(0, "utf8"));
	const out = { files: [] };
	for (const f of input) {
		const sf = ts.createSourceFile(f.path, f.content, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
		const diags = (sf.parseDiagnostics || []).map((d) => {
			const at = sf.getLineAndCharacterOfPosition(d.start || 0);
			return { line: at.line + 1, col: at.character + 1, message: ts.flattenDiagnosticMessageText(d.messageText, "\n") };
		});
		out.files.push({ path: f.path, diagnostics: diags, statements: sf.statements.map((s) => statement(sf, s)) });
	}
	process.stdout.write(JSON.stringify(out));
}

function position(sf, node) {
	const at = sf.getLineAndCharacterOfPosition(node.getStart(sf));
	return { line: at.line + 1, col: at.character + 1 };
}

// docs is the text of each /** */ comment before a node.
function docs(sf, node) {
	const ranges = ts.getLeadingCommentRanges(sf.text, node.getFullStart()) || [];
	return ranges.map((r) => sf.text.slice(r.pos, r.end)).filter((t) => t.startsWith("/**") && t !== "/**/");
}

function modifiers(node) {
	return (ts.canHaveModifiers(node) && ts.getModifiers(node)) || [];
}

function has(node, kind) {
	return modifiers(node).some((m) => m.kind === kind);
}

function statement(sf, s) {
	const out = {
		kind: "other",
		syntax: ts.SyntaxKind[s.kind],
		...position(sf, s),
		docs: docs(sf, s),
		exported: has(s, ts.SyntaxKind.ExportKeyword),
		declare: has(s, ts.SyntaxKind.DeclareKeyword),
	};
	if (s.name && ts.isIdentifier(s.name)) {
		out.name = s.name.text;
	}
	if (ts.isInterfaceDeclaration(s)) {
		out.kind = "interface";
		out.params = (s.typeParameters || []).length;
		out.heritage = (s.heritageClauses || []).length > 0;
		out.members = s.members.map((m) => member(sf, m));
	} else if (ts.isTypeAliasDeclaration(s)) {
		out.kind = "type";
		out.params = (s.typeParameters || []).length;
		out.type = type(sf, s.type);
	} else if (ts.isEnumDeclaration(s)) {
		out.kind = "enum";
		out.members = s.members.map((m) => ({
			kind: "member",
			name: ts.isIdentifier(m.name) || ts.isStringLiteral(m.name) ? m.name.text : m.name.getText(sf),
			...position(sf, m),
			docs: docs(sf, m),
			init: m.initializer ? literal(sf, m.initializer) : undefined,
		}));
	} else if (ts.isExportDeclaration(s) && !s.moduleSpecifier && s.exportClause && ts.isNamedExports(s.exportClause) && s.exportClause.elements.length === 0) {
		out.kind = "empty-export";
	}
	return out;
}

function member(sf, m) {
	const out = { kind: "other", syntax: ts.SyntaxKind[m.kind], ...position(sf, m), docs: docs(sf, m) };
	if (ts.isPropertySignature(m) && m.name && (ts.isIdentifier(m.name) || ts.isStringLiteral(m.name))) {
		out.kind = "property";
		out.name = m.name.text;
		out.quoted = ts.isStringLiteral(m.name);
		out.optional = !!m.questionToken;
		out.readonly = has(m, ts.SyntaxKind.ReadonlyKeyword);
		out.type = m.type ? type(sf, m.type) : { k: "other", text: "" };
	}
	return out;
}

const keywords = {
	[ts.SyntaxKind.StringKeyword]: "string",
	[ts.SyntaxKind.NumberKeyword]: "number",
	[ts.SyntaxKind.BooleanKeyword]: "boolean",
	[ts.SyntaxKind.NeverKeyword]: "never",
};

function type(sf, t) {
	while (ts.isParenthesizedTypeNode(t)) {
		t = t.type;
	}
	if (keywords[t.kind]) {
		return { k: keywords[t.kind] };
	}
	if (ts.isArrayTypeNode(t)) {
		return { k: "array", elem: type(sf, t.elementType) };
	}
	if (ts.isUnionTypeNode(t)) {
		return { k: "union", types: t.types.map((u) => type(sf, u)) };
	}
	if (ts.isTypeReferenceNode(t)) {
		return { k: "ref", name: t.typeName.getText(sf), args: (t.typeArguments || []).map((a) => type(sf, a)) };
	}
	if (ts.isLiteralTypeNode(t)) {
		if (t.literal.kind === ts.SyntaxKind.NullKeyword) {
			return { k: "null" };
		}
		const lit = literal(sf, t.literal);
		if (lit) {
			return { k: "lit", ...lit };
		}
	}
	return { k: "other", text: t.getText(sf) };
}

// literal is a string, number, or boolean literal, or undefined.
function literal(sf, e) {
	if (ts.isStringLiteral(e) || ts.isNoSubstitutionTemplateLiteral(e)) {
		return { str: e.text };
	}
	if (ts.isNumericLiteral(e)) {
		return { num: e.text };
	}
	if (ts.isPrefixUnaryExpression(e) && e.operator === ts.SyntaxKind.MinusToken && ts.isNumericLiteral(e.operand)) {
		return { num: "-" + e.operand.text };
	}
	if (e.kind === ts.SyntaxKind.TrueKeyword || e.kind === ts.SyntaxKind.FalseKeyword) {
		return { bool: e.kind === ts.SyntaxKind.TrueKeyword };
	}
	return undefined;
}

main();
