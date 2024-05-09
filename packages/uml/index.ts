import * as tdl from '@unmango/tdl-es';
import { Readable, Writable } from 'node:stream';

export interface Converter {
	from(reader: Readable): Promise<tdl.Spec>;
}

export interface Generator {
	gen(spec: tdl.Spec, writer: Writable): Promise<void>;
}

export const SUPPORTED_MIME_TYPES = [
	'application/json',
	'application/x-protobuf',
	'application/protobuf',
	'application/vnd.google.protobuf',
] as const;

export type SupportedMimeTypeTuple = typeof SUPPORTED_MIME_TYPES;
export type SupportedMimeType = SupportedMimeTypeTuple[number];

export function read(data: Uint8Array, type?: SupportedMimeType): tdl.Spec {
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
