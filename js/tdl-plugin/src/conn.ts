import type { DescMessage, MessageShape } from "@bufbuild/protobuf";
import { fromBinary, toBinary } from "@bufbuild/protobuf";

import { ProtocolError, UnexpectedEofError } from "./error.js";

/** The wire framing {@link Conn} implements. */
export const FRAMING_VERSION = 1;

/** The protobuf package the model is encoded with. */
export const IR_VERSION = "tdl.ir.v1";

/**
 * Bounds a single message. A length prefix over it is refused before
 * anything is allocated.
 */
export const MAX_MESSAGE_SIZE = 64 << 20;

// A varint holding a 64-bit length is at most ten bytes.
const MAX_VARINT_LENGTH = 10;

/** Where a connection writes: a node:stream Writable, or anything shaped like one. */
export interface Output {
	write(chunk: Uint8Array, callback: (error?: Error | null) => void): unknown;
}

/**
 * One end of a plugin connection. Each message is framed as a varint byte
 * count followed by the encoded message.
 */
export class Conn {
	readonly #output: Output;
	readonly #input: AsyncIterator<Uint8Array>;
	#chunk: Uint8Array = new Uint8Array(0);
	#offset = 0;
	#done = false;

	/**
	 * Returns a connection reading from input, such as a node:stream
	 * Readable, and writing to output.
	 */
	constructor(input: AsyncIterable<Uint8Array>, output: Output) {
		this.#input = input[Symbol.asyncIterator]();
		this.#output = output;
	}

	/** Encodes a message and writes it with its length prefix. */
	async send<Desc extends DescMessage>(schema: Desc, message: MessageShape<Desc>): Promise<void> {
		const body = toBinary(schema, message);
		if (body.length > MAX_MESSAGE_SIZE) {
			throw new ProtocolError(`message exceeds the maximum size: ${body.length} bytes`);
		}

		const prefix: number[] = [];
		let size = body.length;
		while (size >= 0x80) {
			prefix.push((size & 0x7f) | 0x80);
			size = Math.floor(size / 0x80);
		}
		prefix.push(size);

		const frame = new Uint8Array(prefix.length + body.length);
		frame.set(prefix);
		frame.set(body, prefix.length);
		await new Promise<void>((resolve, reject) => {
			this.#output.write(frame, (error) => (error ? reject(error) : resolve()));
		});
	}

	/**
	 * Reads one message, or returns undefined when the stream ends between
	 * messages. A stream that ends part way through one throws an
	 * {@link UnexpectedEofError}.
	 */
	async recv<Desc extends DescMessage>(schema: Desc): Promise<MessageShape<Desc> | undefined> {
		const size = await this.#readLength();
		if (size === undefined) {
			return undefined;
		}
		if (size > MAX_MESSAGE_SIZE) {
			throw new ProtocolError(`message exceeds the maximum size: ${size} bytes`);
		}

		const body = new Uint8Array(size);
		let n = 0;
		while (n < size) {
			if (!(await this.#fill())) {
				throw new UnexpectedEofError("the stream ended inside a message");
			}
			const take = Math.min(size - n, this.#chunk.length - this.#offset);
			body.set(this.#chunk.subarray(this.#offset, this.#offset + take), n);
			this.#offset += take;
			n += take;
		}
		return fromBinary(schema, body);
	}

	// Reads a varint, or returns undefined at the end of the stream before
	// its first byte. A length is bounded by MAX_MESSAGE_SIZE, so a number
	// holds every one worth reading; a longer varint is refused.
	async #readLength(): Promise<number | undefined> {
		let size = 0;
		for (let i = 0; i < MAX_VARINT_LENGTH; i++) {
			if (!(await this.#fill())) {
				if (i === 0) {
					return undefined;
				}
				throw new UnexpectedEofError("the stream ended inside a length prefix");
			}
			const b = this.#chunk[this.#offset++] as number;
			if (i === MAX_VARINT_LENGTH - 1 && b > 1) {
				break;
			}
			size += (b & 0x7f) * 2 ** (7 * i);
			if (b < 0x80) {
				return size;
			}
		}
		throw new ProtocolError("reading length: the varint overflows 64 bits");
	}

	// Makes at least one unread byte available, or returns false at the end
	// of the stream.
	async #fill(): Promise<boolean> {
		while (this.#offset >= this.#chunk.length) {
			if (this.#done) {
				return false;
			}
			const next = await this.#input.next();
			if (next.done) {
				this.#done = true;
				return false;
			}
			this.#chunk = next.value;
			this.#offset = 0;
		}
		return true;
	}
}
