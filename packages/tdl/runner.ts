import { Spec } from '@unmango/tdl-es';
import type { ArrayBufferSink, BunFile } from 'bun';
import type { Runner } from './types';

export class CliRunner implements Runner<Spec> {
	constructor(private path: string) {}

	async from(reader: BunFile): Promise<Spec> {
		const input = await reader.arrayBuffer();
		const proc = Bun.spawn([this.path, 'from'], {
			stdin: new Uint8Array(input),
		});
		const data = await Bun.readableStreamToBytes(proc.stdout);
		return Spec.fromBinary(data);
	}

	async gen(spec: Spec, writer: ArrayBufferSink): Promise<void> {
		const proc = Bun.spawn([this.path, 'gen'], {
			stdin: spec.toBinary(),
		});
		const data = await Bun.readableStreamToBytes(proc.stdout);
		writer.write(data);
	}
}
