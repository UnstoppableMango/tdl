import { gen as generate } from '@unmango/2ts';
import * as tdl from '@unmango/tdl';
import { ArrayBufferSink } from 'bun';

export async function gen(type?: tdl.SupportedMimeType): Promise<void> {
	const buffer = await Bun.stdin.arrayBuffer();
	const spec = tdl.read(new Uint8Array(buffer), type);
	const sink = new ArrayBufferSink();
	await generate(spec, sink);
	await Bun.write(Bun.stdout, sink.end());
}
