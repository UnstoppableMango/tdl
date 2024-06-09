import { Spec } from '@unmango/tdl-es';
import type { Runner } from '.';

export const cliRunner: Runner = {
	async from(reader): Promise<Spec> {
		const input = await reader.arrayBuffer();
		const data = new Uint8Array(input);
		return Spec.fromBinary(data);
	},
	async gen(spec, writer): Promise<void> {
		const data = spec.toBinary();
		await Bun.write(writer, data);
	}
};
