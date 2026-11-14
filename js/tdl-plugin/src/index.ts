/**
 * The TDL plugin SDK: what a backend written in TypeScript builds on.
 *
 * It holds the wire framing, the handshake, a {@link Session} over a request
 * and the analysis tdl sends with it, positioned diagnostics with loss codes,
 * the cascade, and case styles. Like the other SDKs, it computes no fact the
 * analysis carries, and refuses a request without one.
 *
 * A plugin's entry point is {@link run} over its {@link Backend}:
 *
 * ```ts
 * await run(myBackend);
 * ```
 *
 * @module
 */

export { type Backend, type Description, directivesFor, ON_MEMBERS, reply } from "./backend.js";
export {
	camel,
	fromScreaming,
	lastSegment,
	pascal,
	type Style,
	screamingSnake,
	snake,
	words,
} from "./case.js";
export { Conn, FRAMING_VERSION, IR_VERSION, MAX_MESSAGE_SIZE, type Output } from "./conn.js";
export { ProtocolError, UnexpectedEofError } from "./error.js";
/** `tdl.ir.v1`: the resolved model, generated from `proto/` by protobuf-es (`make generate`). */
export * as ir from "./generated/tdl/ir/v1/ir_pb.js";
/** `tdl.plugin.v1`: the plugin protocol. */
export * as protocol from "./generated/tdl/plugin/v1/plugin_pb.js";
export { invalid, LOSS_CODES, type LossCode, loss, lossy, type Problem, unsupported } from "./problem.js";
export { run, serve } from "./server.js";
export { declares, deprecated, doc, fields, params, Session, Skipped, variants } from "./session.js";
