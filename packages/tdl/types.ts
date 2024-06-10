import type { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';

type In = BunFile;
type Out = BunFile;

export type From<T = Spec> = {
	(reader: In): Promise<T>;
};

export type Gen<T = Spec> = {
	(spec: T, writer: Out): Promise<void>;
};

export interface Runner<T = Spec> {
	from: From<T>;
	gen: Gen<T>;
}

// TODO: I'm thinking all abstract all of this into a single pipe type
export type Decoder<T> = {
	(data: T, writer: In): Promise<void>;
}

export type Encoder<T> = {
	(spec: Out): Promise<T>;
}

export interface Serdes<T> {
	decode: Decoder<T>;
	encode: Encoder<T>;
}
