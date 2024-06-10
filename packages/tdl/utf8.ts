import type { ArrayBufferSink, BunFile } from 'bun';
import type { Decoder, Encoder, Serdes } from './types';

export const decode: Decoder<string> = (data: string, writer: ArrayBufferSink): Promise<void> => {
	const result = new TextEncoder().encode(data);
	writer.write(result);
	return Promise.resolve();
};
export const encode: Encoder<string> = async (spec: BunFile): Promise<string> => {
	const data = await spec.arrayBuffer();
	return new TextDecoder().decode(data);
};

export class Utf8Serdes implements Serdes<string> {
	decode: Decoder<string> = decode;
	encode: Encoder<string> = encode;
}
