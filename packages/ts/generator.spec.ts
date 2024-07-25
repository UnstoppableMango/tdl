import { Spec } from '@unmango/tdl-es';
import { ArrayBufferSink } from 'bun';
import { describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import * as YAML from 'yaml';
import { gen } from './generator';
import { tests } from './testdata';

const arbSpec = () =>
	fc.gen().map(g =>
		new Spec({
			name: g(fc.string),
			types: {
				test: {
					fields: {
						a: { type: 'string' },
						b: { type: 'boolean' },
						c: { type: 'number' },
					},
				},
			},
		})
	);

describe('gen', () => {
	it.each(tests)('should generate %p', async (_, source, target) => {
		const spec = new Spec(YAML.parse(source));
		expect(spec).not.toBeNull();

		const buf = new ArrayBufferSink();
		await gen(spec, buf);
		const decoder = new TextDecoder();
		const actual = decoder.decode(buf.end());

		expect(actual).not.toBeNull();
		expect(actual).toEqual(target);
	});

	it('should be deterministic', async () => {
		await fc.assert(fc.asyncProperty(arbSpec(), async (spec) => {
			const bufA = new ArrayBufferSink();
			await gen(spec, bufA);
			const decoder = new TextDecoder();
			const a = decoder.decode(bufA.end());

			const bufB = new ArrayBufferSink();
			await gen(spec, bufB);
			const b = decoder.decode(bufB.end());

			expect(a).not.toBeNull();
			expect(b).not.toBeNull();
			expect(b).toEqual(a);
		}), { numRuns: 1_000 });
	});
});
