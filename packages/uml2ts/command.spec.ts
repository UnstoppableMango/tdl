import type { SupportedMediaType } from '@unmango/tdl';
import { Spec } from '@unmango/tdl-es';
import { afterAll, beforeAll, describe, expect, it } from 'bun:test';
import fc from 'fast-check';
import fs from 'node:fs/promises';
import path from 'node:path';

const binPath = path.join(
	__dirname,
	'dist',
	'uml2ts_test',
);

const ensureClean = async () => {
	if (await fs.exists(binPath)) {
		await fs.unlink(binPath);
	}
};

const arbSpec = () =>
	fc.gen().map(g =>
		new Spec({
			name: g(fc.string),
			types: {
				test: {
					fields: {
						a: { type: 'string' },
						b: { type: 'boolean' },
						c: { type: 'number' },
					},
				},
			},
		})
	);

beforeAll(async () => {
	await ensureClean();
	const proc = Bun.spawn([
		'bun',
		'build',
		'index.ts',
		'--compile',
		'--outfile',
		binPath,
	], { cwd: __dirname });
	await proc.exited;
});

afterAll(ensureClean);

describe('gen', () => {
	it.each<SupportedMediaType>([
		'application/protobuf',
		'application/x-protobuf',
		'application/vnd.google.protobuf',
	])('should read %s data', async (mediaType) => {
		const name = 'testType';
		const spec = new Spec({ types: { [name]: {} } });
		const bytes = spec.toBinary();

		const proc = Bun.spawn([binPath, 'gen', '--type', mediaType], {
			stdin: new Blob([bytes]),
		});

		const actual = await Bun.readableStreamToText(proc.stdout);
		expect(actual).toEqual(`export interface ${name} {\n}\n`);
	});

	it('should read json data', async () => {
		const name = 'testType';
		const spec = new Spec({ types: { [name]: {} } });
		const json = spec.toJsonString();
		const media: SupportedMediaType = 'application/json';

		const proc = Bun.spawn([binPath, 'gen', '--type', media], {
			stdin: Buffer.from(json, 'utf-8'),
		});

		const actual = await Bun.readableStreamToText(proc.stdout);
		expect(actual).toEqual(`export interface ${name} {\n}\n`);
	});

	it('should be deterministic', async () => {
		const media: SupportedMediaType = 'application/json';

		await fc.assert(fc.asyncProperty(arbSpec(), async (spec) => {
			const json = spec.toJsonString();
			const stdin = Buffer.from(json, 'utf-8')

			const procA = Bun.spawn([binPath, 'gen', '--type', media], {
				stdin,
			});

			const procB = Bun.spawn([binPath, 'gen', '--type', media], {
				stdin,
			});

			const [a, b] = await Promise.all([
				Bun.readableStreamToText(procA.stdout),
				Bun.readableStreamToText(procB.stdout),
			]);

			expect(a).toEqual(b);
		}), { numRuns: 20 });
	});
});
