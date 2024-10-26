import { create, toBinary, toJsonString } from '@bufbuild/protobuf';
import { describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import { read, type SupportedMediaType } from './mediaType';
import { SpecSchema } from './proto/v1alpha1/tdl';

const arbSpec = () => {
	return fc.gen().map(g => {
		return create(SpecSchema, {
			name: g(fc.string),
			description: g(fc.string),
			displayName: g(fc.string),
			source: g(fc.string),
			version: g(fc.string),
		});
	});
};

describe('read', () => {
	it.each<SupportedMediaType>([
		'application/protobuf',
		'application/x-protobuf',
		'application/vnd.google.protobuf',
	])('should read %s data', (type) => {
		fc.assert(fc.property(arbSpec(), (spec): void => {
			const bytes = toBinary(SpecSchema, spec);

			const actual = read(bytes, type);

			expect(actual).toEqual(spec);
		}));
	});

	it('should read json data', () => {
		fc.assert(fc.property(arbSpec(), (spec): void => {
			const json = toJsonString(SpecSchema, spec);
			const bytes = Buffer.from(json, 'utf-8');

			const actual = read(bytes, 'application/json');

			expect(actual).toEqual(spec);
		}));
	});
});
