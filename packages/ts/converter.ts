import * as tdl from '@unmango/tdl-es';
import * as uml from '@unmango/uml';
import { Readable, Writable } from 'node:stream';

export class Converter implements uml.Converter {
	from(reader: Readable): Promise<tdl.Spec> {
		throw new Error('Method not implemented.');
	}
}
