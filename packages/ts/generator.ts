import * as tdl from '@unmango/tdl-es';
import * as uml from '@unmango/uml';
import ts from 'typescript';

export class Generator implements uml.Generator {
	gen(spec: tdl.Spec): Promise<string> {
		const source = ts.createSourceFile('types.d.ts', '', ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
		const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed });
		const result = printer.printList(ts.ListFormat.MultiLine, gen(spec), source);
		return Promise.resolve(result);
	}
}

function gen(spec: tdl.Spec): ts.NodeArray<ts.Node> {
	const types = Object.entries(spec.types).map(x => genType(...x));
	return ts.factory.createNodeArray(types);
}

function genType(name: string, type: tdl.Type): ts.Node {
	const props = Object.entries(type.fields).map(x => genProps(...x));

	return ts.factory.createInterfaceDeclaration(
		[ts.factory.createModifier(ts.SyntaxKind.ExportKeyword)],
		name,
		undefined,
		undefined,
		props,
	);
}

function genProps(name: string, field: tdl.Field): ts.PropertySignature {
	const type = ts.factory.createTypeReferenceNode(field.type);

	return ts.factory.createPropertySignature(
		[ts.factory.createModifier(ts.SyntaxKind.ReadonlyKeyword)],
		name,
		undefined,
		type,
	);
}
