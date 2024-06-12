import * as tdl from '@unmango/tdl';
import type { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';
import ts from 'typescript';

export const from: tdl.From<Spec> = async (reader: BunFile): Promise<Spec> => {
	const data = await reader.arrayBuffer();
	const content = new TextDecoder().decode(data);
	const source = ts.createSourceFile('stdin', content, ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
	source.forEachChild(parseNode);
	return await Promise.reject('not implemented');
};

function parseNode(node: ts.Node): void {
	throw new Error('not implemented');
}
