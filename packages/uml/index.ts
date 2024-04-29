import * as tdl from '@unmango/tdl-ts';
import { Readable, Writable } from 'node:stream';

export interface ConverterFrom {
	from(reader: Readable): Promise<tdl.Spec>;
}

export interface ConverterTo {
	to(spec: tdl.Spec, writer: Writable): Promise<void>;
}

export interface Converter extends ConverterFrom, ConverterTo {}

export interface Generator {
	gen(spec: tdl.Spec, writer: Writable): Promise<void>;
}
