import type { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';

export type From<T = Spec> = {
	(reader: BunFile): Promise<T>;
};

export type Gen<T = Spec> = {
	(spec: T, writer: BunFile): Promise<void>;
};

export interface Runner<T = Spec> {
	from: From<T>;
	gen: Gen<T>;
}
