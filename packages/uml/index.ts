import * as tdl from '@unmango/tdl-ts';

export interface ConverterFrom {
	from(reader: ReadableStream): Promise<tdl.Spec>
}

export interface ConverterTo {
	to(spec: tdl.Spec, writer: WritableStream): Promise<void>
}

export interface Converter extends ConverterFrom, ConverterTo {}

export interface Generator {
	gen(spec: tdl.Spec, writer: WritableStream): Promise<void>
}
