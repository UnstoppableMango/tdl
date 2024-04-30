import * as tdl from '@unmango/tdl-es';
import { describe, expect, it } from 'bun:test';
import { Writable } from 'node:stream';
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
				},
			},
			version: '0.1.0',
		});

		await generator.gen(
			spec,
			new Writable({
				write: (chunk) => sink.write(chunk),
			}),
		);

		const decoder = new TextDecoder();
		const actual = decoder.decode(sink.end());
		expect(actual).not.toBeNull();
		expect(actual).toBeEmpty();
	});
})
