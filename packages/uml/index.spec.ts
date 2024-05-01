import { Spec } from '@unmango/tdl-es';
import { describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import { read, type SupportedMimeType } from './index';

const arbSpec = () =>
	fc.gen().map(g =>
		new Spec({
			name: g(fc.string),
			description: g(fc.string),
			displayName: g(fc.string),
			source: g(fc.string),
			version: g(fc.string),
		})
	);

describe('read', () => {
	it.each<SupportedMimeType>([
		'application/protobuf',
		'application/x-protobuf',
		'application/vnd.google.protobuf',
	])('should read %s data', (type) => {
		fc.assert(fc.property(arbSpec(), (spec): void => {
			const bytes = spec.toBinary();

			const actual = read(bytes, type);

			expect(actual).toEqual(spec);
		}));
	});

	it('should read json data', () => {
		fc.assert(fc.property(arbSpec(), (spec): void => {
			const json = spec.toJsonString();
			const bytes = Buffer.from(json, 'utf-8');

			const actual = read(bytes, 'application/json');

			expect(actual).toEqual(spec);
		}));
	});
});
