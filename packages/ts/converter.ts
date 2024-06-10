import * as tdl from '@unmango/tdl';
import type { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';

export const from: tdl.From<Spec> = async (reader: BunFile): Promise<Spec> => {
	return await Promise.reject('not implemented');
};
