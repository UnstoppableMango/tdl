import { Spec } from '@unmango/tdl-es';
import { ArrayBufferSink } from 'bun';
import { describe, expect, it } from 'bun:test';
import * as YAML from 'yaml';
import { gen } from './generator';
import { tests } from './testdata';

describe('Generator', () => {
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

	it('should work', async () => {
		const spec = new Spec({
			version: '0.1.0',
			name: 'test-name',
			description: 'Some description',
			displayName: 'Test Name',
			source: 'https://github.com/UnstoppableMango/tdl',
			labels: {
				test: 'label',
			},
			types: {
				'test': {
					type: 'string',
					fields: {
						test: {
							type: 'string',
						},
					},
				},
			},
		});

		const buf = new ArrayBufferSink();
		await gen(spec, buf);
		const decoder = new TextDecoder();
		const actual = decoder.decode(buf.end());

		expect(actual).not.toBeNull();
		expect(actual).toEqual(`export interface test {\n    readonly test: string;\n}\n`);
	});
});
