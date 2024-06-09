import { Spec } from '@unmango/tdl-es';

export async function from(): Promise<void> {
	const input = await Bun.stdin.arrayBuffer();
	const data = new Uint8Array(input)
	const spec = Spec.fromBinary(data);
	const result = spec.toBinary();
	await Bun.write(Bun.stdout, result);
}


export async function gen(): Promise<void> {
	const input = await Bun.stdin.arrayBuffer();
	const data = new Uint8Array(input)
	const spec = Spec.fromBinary(data);
	const result = spec.toBinary();
	await Bun.write(Bun.stdout, result);
}
