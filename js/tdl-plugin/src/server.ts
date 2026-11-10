import { create } from "@bufbuild/protobuf";

import { type Backend, reply } from "./backend.js";
import { Conn, FRAMING_VERSION, IR_VERSION } from "./conn.js";
import { ProtocolError, UnexpectedEofError } from "./error.js";
import {
	type Handshake,
	HandshakeReplySchema,
	HandshakeSchema,
	ImportRequestSchema,
	ImportResponseSchema,
	Mode,
	RequestSchema,
	ResponseSchema,
} from "./generated/tdl/plugin/v1/plugin_pb.js";

/**
 * Runs a backend over standard input and output, reading requests until the
 * stream ends, and sets the process's exit code. A plugin's entry point is
 * this and nothing else.
 */
export async function run(backend: Backend): Promise<void> {
	try {
		await serve(backend, new Conn(process.stdin, process.stdout));
	} catch (e) {
		// A backend that fails cannot answer; tdl reports what it wrote.
		process.stderr.write(`plugin: ${e instanceof Error ? e.message : String(e)}\n`);
		process.exitCode = 1;
	}
}

/**
 * Serves one connection: the handshake, then requests until the stream
 * ends. A refused handshake is answered and then thrown as a
 * {@link ProtocolError}.
 */
export async function serve(backend: Backend, conn: Conn): Promise<void> {
	const hello = await conn.recv(HandshakeSchema);
	if (hello === undefined) {
		throw new UnexpectedEofError("the stream ended before the handshake");
	}

	const desc = backend.describe();
	const importing = hello.mode === Mode.IMPORT;
	let refusal = incompatible(hello);
	if (refusal === undefined && importing && !desc.reverse) {
		refusal = `${desc.name} does not import; it generates only`;
	}
	if (refusal !== undefined) {
		await conn.send(HandshakeReplySchema, create(HandshakeReplySchema, { accepted: false, refusal }));
		throw new ProtocolError(refusal);
	}

	await conn.send(HandshakeReplySchema, reply(desc));

	if (importing) {
		for (let req = await conn.recv(ImportRequestSchema); req; req = await conn.recv(ImportRequestSchema)) {
			if (!backend.import) {
				throw new ProtocolError(`${desc.name} does not import; it generates only`);
			}
			await conn.send(ImportResponseSchema, await backend.import(req));
		}
		return;
	}

	// Several requests arrive only in a watch session with a backend that
	// declared reuse.
	for (let req = await conn.recv(RequestSchema); req; req = await conn.recv(RequestSchema)) {
		await conn.send(ResponseSchema, await backend.generate(req));
	}
}

// Says what a plugin needed when it cannot read what the host is about to
// send, or returns undefined when it can.
function incompatible(h: Handshake): string | undefined {
	if (h.framingVersion !== FRAMING_VERSION) {
		return `framing version ${h.framingVersion} is not supported; this plugin speaks ${FRAMING_VERSION}`;
	}
	if (h.irVersion !== IR_VERSION) {
		return `model version "${h.irVersion}" is not supported; this plugin reads "${IR_VERSION}"`;
	}
	return undefined;
}
