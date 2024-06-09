import type { Runner } from '@unmango/tdl';
import { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';

export class Echo implements Runner<Spec> {
	async from(reader: BunFile): Promise<Spec> {
		const input = await reader.arrayBuffer();
		const data = new Uint8Array(input);
		return Spec.fromBinary(data);
	}

	async gen(spec: Spec, writer: BunFile): Promise<void> {
		const data = spec.toBinary();
		await Bun.write(writer, data);
	}
}

export const echo: Runner<Spec> = new Echo();
