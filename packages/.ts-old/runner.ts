import * as tdl from '@unmango/tdl';
import type { Spec } from '@unmango/tdl-es';
import { from } from './converter';
import { gen } from './generator';

export class TypescriptRunner implements tdl.Runner<Spec> {
	from: tdl.From<Spec> = from;
	gen: tdl.Gen<Spec> = gen;
}
