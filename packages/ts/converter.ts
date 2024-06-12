import * as tdl from '@unmango/tdl';
import { Spec, Type } from '@unmango/tdl-es';
import type { BunFile } from 'bun';
import ts from 'typescript';

export const from: tdl.From<Spec> = async (reader: BunFile): Promise<Spec> => {
	const data = await reader.arrayBuffer();
	const content = new TextDecoder().decode(data);
	const source = ts.createSourceFile('stdin', content, ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
	const parts = source.forEachChild(parseNode);
	return await Promise.reject('not implemented');
};

function parseNode(node: ts.Node): Spec {
	switch (node.kind) {
		case ts.SyntaxKind.InterfaceDeclaration:
			return parseInterface(node as ts.InterfaceDeclaration);
	}

	return new Spec();
}

function parseInterface(node: ts.InterfaceDeclaration): Spec {
	const type = new Type();
	const name = node.name.text;

	return new Spec({
		types: {
			[name]: type,
		},
	});
}
