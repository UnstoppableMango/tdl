import { Field, Spec, Type } from '@unmango/tdl-es';
import type { SupportedMimeType } from '@unmango/uml';
import { describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import { gen, type Io } from './command';

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
				const io: Io = {
					stdin: new Blob([bytes]),
					stdout: new Bun.ArrayBufferSink(),
				};

				await gen(io, mime);

				const decoder = new TextDecoder();
				const actual = decoder.decode(io.stdout.end());
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
				const io: Io = {
					stdin: new Blob([Buffer.from(json, 'utf-8')]),
					stdout: new Bun.ArrayBufferSink(),
				};

				await gen(io, 'application/json');

				const decoder = new TextDecoder();
				const actual = decoder.decode(io.stdout.end());
				expect(actual).toEqual(`export interface ${name} {\n}\n`);
			},
		));
	});
});
