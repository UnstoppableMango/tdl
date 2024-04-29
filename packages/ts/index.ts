import * as tdl from '@unmango/tdl-ts';
import * as uml from '@unmango/uml';

class Converter implements uml.Converter {
	from(reader: ReadableStream): Promise<tdl.Spec> {
		throw new Error('Method not implemented.');
	}
	to(spec: tdl.Spec, writer: WritableStream): Promise<void> {
		throw new Error('Method not implemented.');
	}
}

class Generator implements uml.Generator {
	gen(spec: tdl.Spec, writer: WritableStream): Promise<void> {
		throw new Error('Method not implemented.');
	}
}

export const converter = new Converter();
export const generator = new Generator();
