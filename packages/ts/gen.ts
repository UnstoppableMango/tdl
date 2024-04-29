import type { Spec } from '@unmango/tdl-ts';
import ts from 'typescript';

export function gen(spec: Spec): ts.NodeArray<ts.Node> {
	const node = ts.factory.createInterfaceDeclaration([], 'Test', undefined, undefined, []);
	return ts.factory.createNodeArray([node]);
}
