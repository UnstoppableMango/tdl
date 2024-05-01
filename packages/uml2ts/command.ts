import { generator } from '@unmango/2ts';
import * as uml from '@unmango/uml';
import type { BunFile } from 'bun';

export interface Io {
	stdin: BunFile;
	stdout: BunFile;
}

export async function gen({ stdin, stdout }: Io, type?: uml.SupportedMimeType): Promise<void> {
	if (!stdin.readable) {
		throw new Error('stdin was not readable');
	}

	const buffer = await stdin.arrayBuffer();
	const spec = uml.read(new Uint8Array(buffer), type);
	const ts = await generator.gen(spec);

	stdout.writer().write(ts);
}
