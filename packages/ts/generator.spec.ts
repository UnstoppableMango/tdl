import * as tdl from '@unmango/tdl-es';
import { describe, expect, it } from 'bun:test';
import { generator } from '.';

describe('Generator', () => {
	it('should work', async () => {
		const spec = new tdl.Spec({
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
						}
					},
				},
			},
			version: '0.1.0',
		});

		const actual = await generator.gen(spec);

		expect(actual).not.toBeNull();
		expect(actual).toEqual(`export interface test {\n    readonly test: string;\n}\n`);
	});
});
