import * as tdl from '@unmango/tdl-ts';
import * as uml from '@unmango/uml';
import { Writable } from 'node:stream';
import ts from 'typescript';

export class Generator implements uml.Generator {
	gen(spec: tdl.Spec, writer: Writable): Promise<void> {
		const source = ts.createSourceFile('types.d.ts', '', ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
		const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed });
		const result = printer.printList(ts.ListFormat.MultiLine, gen(spec), source);

		const encoding: BufferEncoding = 'utf-8';
		const bytes = Buffer.from(result, encoding);
		writer.write(bytes, encoding);
		writer.end();

		return Promise.resolve();
	}
}

function gen(spec: tdl.Spec): ts.NodeArray<ts.Node> {
	const types = Object.entries(spec.types).map(x => genType(...x));
	return ts.factory.createNodeArray(types);
}

function genType(name: string, type: tdl.Type): ts.Node {
	return ts.factory.createInterfaceDeclaration([], name, undefined, undefined, []);
}
