import * as tdl from '@unmango/tdl-es';
import { Readable, Writable } from 'node:stream';

export interface ConverterFrom {
	from(reader: Readable): Promise<tdl.Spec>;
}

export interface ConverterTo {
	to(spec: tdl.Spec, writer: Writable): Promise<void>;
}

export interface Converter extends ConverterFrom, ConverterTo {}

export interface Generator {
	gen(spec: tdl.Spec): Promise<string>;
}

export function read(data: Uint8Array, type?: string): tdl.Spec {
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
		case '':
			return tdl.Spec.fromBinary(data);
		default:
			throw new Error('unrecognized media type');
	}
}
