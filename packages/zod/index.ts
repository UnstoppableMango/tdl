import type { Field, Spec, Type } from '@unmango/tdl/v1alpha1/tdl';
import { create } from '@unmango/tdl/spec';
import { ZodObject, ZodType, type ZodSchema } from 'zod';

type Schema = Record<string, ZodSchema>;

export function parse(schema: Schema): Spec {
	if (schema instanceof ZodObject) {
		return create({
			name: schema._def.description,
		});
	}

	return create();
}

export function parseObject(name: string, schema: ZodType): Spec {
	return create({ name });
}
