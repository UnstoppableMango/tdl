import { Spec } from '@unmango/tdl-es';
import { ArrayBufferSink } from 'bun';
import { describe, expect, it } from 'bun:test';
import * as YAML from 'yaml';
import { tests } from './testdata';
import { from } from './converter';

describe('from', () => {
	it.each(tests)('should generate %p', async (_, source, target) => {
		const expected = new Spec(YAML.parse(source));
		expect(expected).not.toBeNull();

		const data = new TextEncoder().encode(target);
		const result = await from(data);
		const decoder = new TextDecoder();
		const actual = decoder.decode(buf.end());

		expect(actual).not.toBeNull();
		expect(actual).toEqual(target);
	});
});
