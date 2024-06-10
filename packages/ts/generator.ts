import type { Gen } from '@unmango/tdl';
import type { Field, Spec, Type } from '@unmango/tdl-es';
import type { BunFile } from 'bun';
import ts from 'typescript';

export const gen: Gen<Spec> = async (spec: Spec, writer: BunFile): Promise<void> => {
	const source = ts.createSourceFile('types.d.ts', '', ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
	const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed });
	const result = printer.printList(ts.ListFormat.MultiLine, genNodes(spec), source);
	await Bun.write(writer, result);
};

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
		[ts.factory.createModifier(ts.SyntaxKind.ReadonlyKeyword)],
		name,
		undefined,
		type,
	);
}
