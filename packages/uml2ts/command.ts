import { generator } from '@unmango/2ts';
import * as uml from '@unmango/uml';

export async function gen(type?: uml.SupportedMimeType): Promise<void> {
	const buffer = await Bun.stdin.arrayBuffer();
	const spec = uml.read(new Uint8Array(buffer), type);
	const ts = await generator.gen(spec);
	await Bun.write(Bun.stdout, ts);
}
