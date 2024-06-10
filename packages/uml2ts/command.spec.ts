import { Spec } from '@unmango/tdl-es';
import type { SupportedMimeType } from '@unmango/tdl';
import { afterAll, beforeAll, describe, expect, it } from 'bun:test';
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
	it.each<SupportedMimeType>([
		'application/protobuf',
		'application/x-protobuf',
		'application/vnd.google.protobuf',
	])('should read %s data', async (mime) => {
		const name = 'testType';
		const spec = new Spec({ types: { [name]: {} } });
		const bytes = spec.toBinary();

		const proc = Bun.spawn([binPath, 'gen', '--type', mime], {
			stdin: new Blob([bytes]),
		});

		const actual = await Bun.readableStreamToText(proc.stdout);
		expect(actual).toEqual(`export interface ${name} {\n}\n`);
	});

	it('should read json data', async () => {
		const name = 'testType';
		const spec = new Spec({ types: { [name]: {} } });
		const json = spec.toJsonString();
		const mime: SupportedMimeType = 'application/json';

		const proc = Bun.spawn([binPath, 'gen', '--type', mime], {
			stdin: Buffer.from(json, 'utf-8'),
		});

		const actual = await Bun.readableStreamToText(proc.stdout);
		expect(actual).toEqual(`export interface ${name} {\n}\n`);
	});
});
