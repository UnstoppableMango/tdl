import { generator } from '@unmango/2ts';
import * as uml from '@unmango/uml';
import type { ArrayBufferSink } from 'bun';

export interface Io {
	stdin: Blob;
	stdout: ArrayBufferSink;
}

export async function gen({ stdin, stdout }: Io, type?: uml.SupportedMimeType): Promise<void> {
	const buffer = await stdin.arrayBuffer();
	const spec = uml.read(new Uint8Array(buffer), type);
	const ts = await generator.gen(spec);

	stdout.write(ts);
}
