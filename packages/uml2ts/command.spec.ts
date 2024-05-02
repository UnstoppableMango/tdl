import { Field, Spec, Type } from '@unmango/tdl-es';
import type { SupportedMimeType } from '@unmango/uml';
import { beforeAll, describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import path from 'node:path';

const arbField = () =>
	fc.gen().map(g =>
		new Field({
			type: g(fc.string),
		})
	);

const arbType = (fields: Record<string, fc.Arbitrary<Field>>) =>
	fc.gen().map(g =>
		new Type({
			fields: g(() => fc.record(fields)),
		})
	);

// TODO: Using `fc.string()` for a `Type` name is flaky because we haven't properly defined what a `TypeName` can be yet.
//       Would probably be better to have a custom `TypeName` arb and/or actually add `TypName` to the UMl spec.

const binPath = path.join(
	__dirname,
	'dist',
	'uml2ts_test'
);

beforeAll(() => {
	Bun.spawn([
		'bun',
		'build',
		'index.ts',
		'--compile',
		'--outfile',
		binPath,
	], { cwd: __dirname });
});

describe('gen', () => {
	it.each<SupportedMimeType>([
		'application/protobuf',
		'application/x-protobuf',
		'application/vnd.google.protobuf',
	])('should read %s data', async (mime) => {
		await fc.assert(fc.asyncProperty(
			fc.tuple(fc.string(), arbType({})),
			async ([name, type]): Promise<void> => {
				const spec = new Spec({ types: { [name]: type } });
				const bytes = spec.toBinary();

				const proc = Bun.spawn([binPath, 'gen', '--type', mime], {
					stdin: new Blob([bytes]),
				});

				const actual = await Bun.readableStreamToText(proc.stdout);
				expect(actual).toEqual(`export interface ${name} {\n}\n`);
			},
		));
	});

	it('should read json data', async () => {
		await fc.assert(fc.asyncProperty(
			fc.tuple(fc.string(), arbType({})),
			async ([name, type]): Promise<void> => {
				const spec = new Spec({ types: { [name]: type } });
				const json = spec.toJsonString();
				const mime: SupportedMimeType = 'application/json';

				const proc = Bun.spawn([binPath, 'gen', '--type', mime], {
					stdin: Buffer.from(json, 'utf-8'),
				});

				const actual = await Bun.readableStreamToText(proc.stdout);
				expect(actual).toEqual(`export interface ${name} {\n}\n`);
			},
		));
	});
});
