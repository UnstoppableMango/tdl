import type { Position } from "./generated/tdl/ir/v1/ir_pb.js";

/**
 * Something a backend reports about the model, at a position.
 *
 * An `unsupported` problem is a shape the backend cannot express. It reaches
 * the user as a warning, carrying a code from {@link loss} when its omission
 * is one the user may choose to allow, or an empty code otherwise.
 *
 * An `invalid` problem is a directive the user got wrong, such as two fields
 * pinned to one number. It reaches the user as an error, so the host writes
 * nothing.
 */
export type Problem =
	| { kind: "unsupported"; message: string; position: Position | undefined; code: LossCode | "" }
	| { kind: "invalid"; message: string; position: Position | undefined };

/** An unsupported problem with no loss code. */
export function unsupported(message: string, position?: Position): Problem {
	return { kind: "unsupported", message, position, code: "" };
}

/** An unsupported problem carrying a loss code. */
export function lossy(code: LossCode, message: string, position?: Position): Problem {
	return { kind: "unsupported", message, position, code };
}

/** An invalid problem. */
export function invalid(message: string, position?: Position): Problem {
	return { kind: "invalid", message, position };
}

/**
 * Loss codes name a fact a conversion cannot carry, in either direction. A
 * warning carrying one can be silenced in `tdl.toml` or with
 * `--allow-lossy`. `docs/architecture/reverse.md` describes each.
 */
export const loss = {
	PRIMITIVE: "lossy.primitive",
	NEWTYPE: "lossy.newtype",
	COLLECTION: "lossy.collection",
	OPTIONAL: "lossy.optional",
	STRUCT_KIND: "lossy.struct-kind",
	INCLUDE: "lossy.include",
	ALIAS: "lossy.alias",
	CONSTRAINT: "lossy.constraint",
	OWNED: "lossy.owned",
	DEFAULT: "lossy.default",
	KEY: "lossy.key",
	NAME: "lossy.name",
	NUMBER: "lossy.number",
	ORDER: "lossy.order",
	DOC: "lossy.doc",
	GENERIC: "lossy.generic",
	CLASS: "lossy.class",
	UNIT: "lossy.unit",
	UNSUPPORTED: "lossy.unsupported",
} as const;

/** A loss code. */
export type LossCode = (typeof loss)[keyof typeof loss];

/** Every loss code, in the order the design lists them. */
export const LOSS_CODES: readonly LossCode[] = Object.values(loss);
