import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { create, equals, toBinary } from "@bufbuild/protobuf";

import { FRAMING_VERSION, IR_VERSION, MAX_MESSAGE_SIZE } from "../src/conn.js";
import { ProtocolError, UnexpectedEofError } from "../src/error.js";
import { HandshakeSchema, ResponseSchema } from "../src/generated/tdl/plugin/v1/plugin_pb.js";
import { framed, reading } from "./helpers.js";

const hello = create(HandshakeSchema, { framingVersion: FRAMING_VERSION, irVersion: IR_VERSION });

describe("Conn", () => {
	test("round trips messages, across chunk boundaries", async () => {
		const bytes = await framed(async (c) => {
			await c.send(HandshakeSchema, hello);
			await c.send(ResponseSchema, create(ResponseSchema));
		});

		for (const size of [bytes.length, 1, 3]) {
			const conn = reading(bytes, undefined, size);
			const got = await conn.recv(HandshakeSchema);
			assert.ok(got && equals(HandshakeSchema, got, hello));
			assert.ok(await conn.recv(ResponseSchema));
			assert.equal(await conn.recv(ResponseSchema), undefined);
		}
	});

	test("prefixes the length as a varint", async () => {
		const big = create(ResponseSchema, { post: ["x".repeat(200)] });
		const body = toBinary(ResponseSchema, big);

		const bytes = await framed((c) => c.send(ResponseSchema, big));

		assert.equal(bytes[0], (body.length & 0x7f) | 0x80);
		assert.equal(bytes[1], body.length >> 7);
		assert.deepEqual(bytes.subarray(2), body);
	});

	test("a stream ending inside a prefix is an unexpected end", async () => {
		await assert.rejects(reading(new Uint8Array([0x80])).recv(ResponseSchema), UnexpectedEofError);
	});

	test("a stream ending inside a body is an unexpected end", async () => {
		await assert.rejects(reading(new Uint8Array([5, 1])).recv(ResponseSchema), UnexpectedEofError);
	});

	test("refuses a length over the maximum", async () => {
		// 64 MiB + 1 as a varint.
		const prefix: number[] = [];
		let n = MAX_MESSAGE_SIZE + 1;
		while (n >= 0x80) {
			prefix.push((n & 0x7f) | 0x80);
			n >>>= 7;
		}
		prefix.push(n);

		await assert.rejects(reading(new Uint8Array(prefix)).recv(ResponseSchema), (e: unknown) => {
			assert.ok(e instanceof ProtocolError);
			assert.match(e.message, /maximum size/);
			return true;
		});
	});

	test("refuses a varint over 64 bits", async () => {
		await assert.rejects(reading(new Uint8Array(10).fill(0xff)).recv(ResponseSchema), (e: unknown) => {
			assert.ok(e instanceof ProtocolError);
			assert.match(e.message, /overflows/);
			return true;
		});
	});
});
