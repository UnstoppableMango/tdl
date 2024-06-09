import { Spec } from '@unmango/tdl-es';
import type { BunFile } from 'bun';
import type { Runner } from '.';

export class CliRunner implements Runner<Spec> {
	constructor(private path: string) {}
	async from(reader: BunFile): Promise<Spec> {
		const input = await reader.arrayBuffer();
		const data = new Uint8Array(input);
		const proc = Bun.spawn([this.path, 'from']);
		await Bun.write(proc.stdin, data);
		return Spec.fromBinary(data);
	}
	async gen(spec: Spec, writer: BunFile): Promise<void> {
		const data = spec.toBinary();
		await Bun.write(writer, data);
	}
}
