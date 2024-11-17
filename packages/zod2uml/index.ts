import { parse, parseObject } from '@unmango/2zod';
import * as fs from 'fs/promises';
import * as path from 'path';
import { z, type ZodTypeAny } from 'zod';

if (process.argv.length !== 3) {
	console.log('path to zod schema definition script is required');
	process.exit(1);
}

const schemaPath = process.argv[2];
if (!await fs.exists(schemaPath)) {
	console.log('not found: %s', schemaPath);
	process.exit(1);
}

const module = z.record(z.string(), z.unknown());
const exports = module.parse(await import(schemaPath));

if (exports.default) {
	if (!isZod(exports.default)) {
		console.log('default export was not a supported zod type');
		process.exit(1);
	}

	const ext = path.extname(schemaPath);
	const name = path.basename(schemaPath, ext);
	const spec = parseObject(name, exports.default);
	console.log(JSON.stringify(spec));
} else {
	const schemas: Record<string, ZodTypeAny> = {};
	for (const [k, v] of Object.entries(exports)) {
		if (!isZod(v)) {
			console.log('exported member %s was not a supported zod type');
			process.exit(1);
		}

		schemas[k] = v;
	}

	const spec = parse(schemas);
	console.log(JSON.stringify(spec));
}

function isZod(x: unknown): x is ZodTypeAny {
	if (!x?.constructor.name) {
		return false;
	}

	return ['ZodObject'].includes(x.constructor.name);
}
