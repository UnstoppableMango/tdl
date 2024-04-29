import * as tdl from '@unmango/tdl-ts';
import * as uml from '@unmango/uml';
import { Writable } from 'node:stream';
import ts from 'typescript';
import { gen } from './gen';

export class Generator implements uml.Generator {
	gen(spec: tdl.Spec, writer: Writable): Promise<void> {
		const nodes = gen(spec);

		const source = ts.createSourceFile('types.d.ts', '', ts.ScriptTarget.ES2019, undefined, ts.ScriptKind.TS);
		const printer = ts.createPrinter({ newLine: ts.NewLineKind.LineFeed });
		const result = printer.printList(ts.ListFormat.MultiLine, nodes, source);

		const encoding: BufferEncoding = 'utf-8';
		const bytes = Buffer.from(result, encoding);
		writer.write(bytes, encoding);
		writer.end();

		return Promise.resolve();
	}
}
