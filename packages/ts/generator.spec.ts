import { create } from '@unmango/tdl/spec';
import { describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import * as YAML from 'yaml';
import { gen } from './generator';
import { tests } from './testdata';

const arbSpec = () => {
	return fc.gen().map(g =>
		create({
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
};

describe('gen', () => {
	it.each(tests)('should generate %p', (_, source, target) => {
		const spec = create(YAML.parse(source));
		expect(spec).not.toBeNull();

		const actual = gen(spec);

		expect(actual).not.toBeNull();
		expect(actual).toEqual(target);
	});

	it('should be deterministic', () => {
		fc.assert(
			fc.property(arbSpec(), (spec) => {
				const a = gen(spec);
				const b = gen(spec);

				expect(a).not.toBeNull();
				expect(b).not.toBeNull();
				expect(b).toEqual(a);
			}),
			{ numRuns: 1_000 },
		);
	});
});
