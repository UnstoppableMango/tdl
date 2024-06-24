import * as tdl from '@unmango/tdl-es';

export const SUPPORTED_MEDIA_TYPES = [
	'application/json',
	'application/x-protobuf',
	'application/protobuf',
	'application/vnd.google.protobuf',
] as const;

export type SupportedMediaTypeTuple = typeof SUPPORTED_MEDIA_TYPES;
export type SupportedMediaType = SupportedMediaTypeTuple[number];

export function read(data: Uint8Array, type?: SupportedMediaType): tdl.Spec {
	switch (type) {
		case 'application/json': {
			const decoder = new TextDecoder();
			const json = decoder.decode(data);
			return tdl.Spec.fromJsonString(json);
		}
		case 'application/x-protobuf':
		case 'application/protobuf':
		case 'application/vnd.google.protobuf':
			return tdl.Spec.fromBinary(data);
		case undefined:
		case null:
			return tdl.Spec.fromBinary(data);
		default:
			throw new Error('unrecognized media type');
	}
}
