import * as tdl from '@unmango/tdl-ts';
import * as uml from '@unmango/uml';
import { Readable, Writable } from 'node:stream';

export class Converter implements uml.Converter {
	from(reader: Readable): Promise<tdl.Spec> {
		throw new Error('Method not implemented.');
	}
	to(spec: tdl.Spec, writer: Writable): Promise<void> {
		throw new Error('Method not implemented.');
	}
}
