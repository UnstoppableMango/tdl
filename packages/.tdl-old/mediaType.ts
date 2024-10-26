import { fromBinary, fromJsonString } from '@bufbuild/protobuf';
import { type Spec, SpecSchema } from './proto/v1alpha1/tdl';

export const SUPPORTED_MEDIA_TYPES = [
	'application/json',
	'application/x-protobuf',
	'application/protobuf',
	'application/vnd.google.protobuf',
] as const;

export type SupportedMediaTypeTuple = typeof SUPPORTED_MEDIA_TYPES;
export type SupportedMediaType = SupportedMediaTypeTuple[number];

export function read(data: Uint8Array, type?: SupportedMediaType): Spec {
	switch (type) {
		case 'application/json': {
			const decoder = new TextDecoder();
			const json = decoder.decode(data);
			return fromJsonString(SpecSchema, json);
		}
		case 'application/x-protobuf':
		case 'application/protobuf':
		case 'application/vnd.google.protobuf':
			return fromBinary(SpecSchema, data);
		case undefined:
		case null:
			return fromBinary(SpecSchema, data);
		default:
			throw new Error('unrecognized media type');
	}
}
