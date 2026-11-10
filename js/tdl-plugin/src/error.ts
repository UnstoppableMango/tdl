/**
 * A failure of the protocol itself, or of a request the SDK cannot serve,
 * rather than a problem with the model, which belongs in a response's
 * diagnostics.
 */
export class ProtocolError extends Error {
	override name = "ProtocolError";
}

/** A stream that ended part way through a message, or before the handshake. */
export class UnexpectedEofError extends Error {
	override name = "UnexpectedEofError";
}
