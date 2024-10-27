import type { Field, Spec, Type } from '@unmango/tdl/v1alpha1/tdl';
import ts from 'typescript';

export function gen(spec: Spec): string {
	const source = ts.createSourceFile('types.d.ts', '', ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
	const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed });
	return printer.printList(ts.ListFormat.MultiLine, genNodes(spec), source);
}

function genNodes(spec: Spec): ts.NodeArray<ts.Node> {
	const types = Object.entries(spec.types).map(x => genType(...x));
	return ts.factory.createNodeArray(types);
}

function genType(name: string, type: Type): ts.Node {
	const props = Object.entries(type.fields).map(x => genProps(...x));

	return ts.factory.createInterfaceDeclaration(
		[ts.factory.createModifier(ts.SyntaxKind.ExportKeyword)],
		name,
		undefined,
		undefined,
		props,
	);
}

function genProps(name: string, field: Field): ts.PropertySignature {
	const type = ts.factory.createTypeReferenceNode(field.type);

	return ts.factory.createPropertySignature(
		// [ts.factory.createModifier(ts.SyntaxKind.ReadonlyKeyword)],
		[],
		name,
		undefined,
		type,
	);
}
