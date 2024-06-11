import { Spec } from '@unmango/tdl-es';
import { ArrayBufferSink } from 'bun';
import { describe, expect, it } from 'bun:test';
import * as YAML from 'yaml';
import { gen } from './generator';
import { tests } from './testdata';

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
});
