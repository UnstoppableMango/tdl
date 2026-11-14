// The debug port answers each request the Go host recorded in testdata/plugin
// with the bytes the Go debug backend answered it with.

import assert from "node:assert/strict";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { describe, test } from "node:test";
import { fileURLToPath } from "node:url";

import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { Conn, FRAMING_VERSION, IR_VERSION, protocol, reply, serve } from "@unmango/tdl-plugin";

import { debug } from "../src/index.js";

function pluginData(): string {
	let dir = dirname(fileURLToPath(import.meta.url));
	while (!existsSync(join(dir, "testdata", "plugin"))) {
		const up = dirname(dir);
		assert.notEqual(up, dir, "no testdata/plugin above the tests");
		dir = up;
	}
	return join(dir, "testdata", "plugin");
}

// The base names of the recorded exchanges.
const exchanges = readdirSync(pluginData())
	.filter((name) => name.endsWith(".request.binpb"))
	.map((name) => name.slice(0, -".request.binpb".length))
	.sort();

const read = (name: string) => new Uint8Array(readFileSync(join(pluginData(), name)));
const request = (exchange: string) => fromBinary(protocol.RequestSchema, read(`${exchange}.request.binpb`));
const response = (exchange: string) => read(`${exchange}.response.binpb`);

// An output collecting what is written to it.
function sink() {
	const chunks: Uint8Array[] = [];
	return {
		write(chunk: Uint8Array, callback: (error?: Error | null) => void) {
			chunks.push(chunk);
			callback();
		},
		bytes: () => new Uint8Array(Buffer.concat(chunks)),
	};
}

async function* yielding(bytes: Uint8Array) {
	yield bytes;
}

// The frames a connection writes for these messages.
async function framed(send: (conn: Conn) => Promise<void>): Promise<Uint8Array> {
	const out = sink();
	await send(new Conn(yielding(new Uint8Array(0)), out));
	return out.bytes();
}

describe("the debug port", () => {
	test("has exchanges to replay", () => {
		assert.ok(exchanges.length > 0);
	});

	for (const exchange of exchanges) {
		test(`answers ${exchange} byte for byte`, async () => {
			const resp = await debug.generate(request(exchange));

			assert.deepEqual(toBinary(protocol.ResponseSchema, resp), response(exchange));
		});

		test(`answers ${exchange} over a connection`, async () => {
			const hello = create(protocol.HandshakeSchema, { framingVersion: FRAMING_VERSION, irVersion: IR_VERSION });
			const input = await framed(async (c) => {
				await c.send(protocol.HandshakeSchema, hello);
				await c.send(protocol.RequestSchema, request(exchange));
			});
			const out = sink();

			await serve(debug, new Conn(yielding(input), out));

			const want = await framed(async (c) => {
				await c.send(protocol.HandshakeReplySchema, reply(debug.describe()));
				await c.send(protocol.ResponseSchema, fromBinary(protocol.ResponseSchema, response(exchange)));
			});
			assert.deepEqual(out.bytes(), want);
		});
	}
});
