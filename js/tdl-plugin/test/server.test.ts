import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { create, equals } from "@bufbuild/protobuf";

import { type Backend, type Description, reply } from "../src/backend.js";
import { Conn, FRAMING_VERSION, IR_VERSION } from "../src/conn.js";
import { ProtocolError, UnexpectedEofError } from "../src/error.js";
import {
	FileSchema,
	type Handshake,
	HandshakeReplySchema,
	HandshakeSchema,
	Mode,
	RequestSchema,
	ResponseSchema,
} from "../src/generated/tdl/plugin/v1/plugin_pb.js";
import { serve } from "../src/server.js";
import { chunked, framed, Sink } from "./helpers.js";

// Answers a request with a file named for its target.
function echo(reverse = false): Backend {
	return {
		describe: (): Description => ({ name: "echo", version: "0.0.1", reverse }),
		generate: (req) => create(ResponseSchema, { files: [create(FileSchema, { path: req.target })] }),
	};
}

const hello = (init: Partial<Handshake> = {}) =>
	create(HandshakeSchema, { framingVersion: FRAMING_VERSION, irVersion: IR_VERSION, ...init });

async function exchange(backend: Backend, input: Uint8Array) {
	const sink = new Sink();
	const result = serve(backend, new Conn(chunked(input), sink)).then(
		() => undefined,
		(e: unknown) => e,
	);
	return { error: await result, out: sink.bytes() };
}

async function replyIn(out: Uint8Array) {
	const got = await new Conn(chunked(out), new Sink()).recv(HandshakeReplySchema);
	assert.ok(got);
	return got;
}

describe("serve", () => {
	test("answers each request after the handshake", async () => {
		const input = await framed(async (c) => {
			await c.send(HandshakeSchema, hello());
			for (const target of ["a", "b"]) {
				await c.send(RequestSchema, create(RequestSchema, { target }));
			}
		});

		const { error, out } = await exchange(echo(), input);

		assert.equal(error, undefined);
		const conn = new Conn(chunked(out), new Sink());
		const got = await conn.recv(HandshakeReplySchema);
		assert.ok(got && equals(HandshakeReplySchema, got, reply(echo().describe())));
		for (const target of ["a", "b"]) {
			assert.equal((await conn.recv(ResponseSchema))?.files[0]?.path, target);
		}
		assert.equal(await conn.recv(ResponseSchema), undefined);
	});

	test("refuses another framing version", async () => {
		const input = await framed((c) => c.send(HandshakeSchema, hello({ framingVersion: 2 })));

		const { error, out } = await exchange(echo(), input);

		const got = await replyIn(out);
		assert.equal(got.accepted, false);
		assert.match(got.refusal, /framing version 2/);
		assert.ok(error instanceof ProtocolError);
		assert.equal(error.message, got.refusal);
	});

	test("refuses another model version", async () => {
		const input = await framed((c) => c.send(HandshakeSchema, hello({ irVersion: "tdl.ir.v2" })));

		const { error, out } = await exchange(echo(), input);

		assert.match((await replyIn(out)).refusal, /"tdl\.ir\.v2"/);
		assert.ok(error instanceof ProtocolError);
	});

	test("refuses import to a backend that only generates", async () => {
		const input = await framed((c) => c.send(HandshakeSchema, hello({ mode: Mode.IMPORT })));

		const { error, out } = await exchange(echo(), input);

		assert.equal((await replyIn(out)).refusal, "echo does not import; it generates only");
		assert.ok(error instanceof ProtocolError);
	});

	test("an empty stream is an unexpected end", async () => {
		const { error, out } = await exchange(echo(), new Uint8Array(0));

		assert.equal(out.length, 0);
		assert.ok(error instanceof UnexpectedEofError);
	});
});
