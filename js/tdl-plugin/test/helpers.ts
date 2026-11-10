// What the tests share: the repository's testdata, and connections over
// bytes in memory.

import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { fromBinary } from "@bufbuild/protobuf";

import { Conn, type Output } from "../src/conn.js";
import { type Request, RequestSchema } from "../src/generated/tdl/plugin/v1/plugin_pb.js";

/** The repository root: the first directory above this file holding testdata/plugin. */
export function root(): string {
	let dir = dirname(fileURLToPath(import.meta.url));
	while (!existsSync(join(dir, "testdata", "plugin"))) {
		const up = dirname(dir);
		if (up === dir) {
			throw new Error(`no testdata/plugin above ${fileURLToPath(import.meta.url)}`);
		}
		dir = up;
	}
	return dir;
}

/** A request recorded in testdata/plugin. */
export function request(exchange: string): Request {
	return fromBinary(RequestSchema, readFileSync(join(root(), "testdata", "plugin", `${exchange}.request.binpb`)));
}

/** An output collecting what is written to it. */
export class Sink implements Output {
	readonly chunks: Uint8Array[] = [];

	write(chunk: Uint8Array, callback: (error?: Error | null) => void): boolean {
		this.chunks.push(chunk);
		callback();
		return true;
	}

	/** Everything written, in one array. */
	bytes(): Uint8Array {
		return concat(this.chunks);
	}
}

/** Joins byte arrays. */
export function concat(parts: readonly Uint8Array[]): Uint8Array {
	const out = new Uint8Array(parts.reduce((n, p) => n + p.length, 0));
	let n = 0;
	for (const p of parts) {
		out.set(p, n);
		n += p.length;
	}
	return out;
}

/** An input yielding bytes in chunks of at most `size`, so reads cross chunk boundaries. */
export async function* chunked(bytes: Uint8Array, size = bytes.length || 1): AsyncGenerator<Uint8Array> {
	for (let i = 0; i < bytes.length; i += size) {
		yield bytes.subarray(i, i + size);
	}
}

/** A connection reading `bytes` and writing to `sink`. */
export function reading(bytes: Uint8Array, sink: Sink = new Sink(), size?: number): Conn {
	return new Conn(chunked(bytes, size), sink);
}

/** The bytes a connection writes when `write` sends to it. */
export async function framed(write: (conn: Conn) => Promise<void>): Promise<Uint8Array> {
	const sink = new Sink();
	await write(new Conn(chunked(new Uint8Array(0)), sink));
	return sink.bytes();
}
