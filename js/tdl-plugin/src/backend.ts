import { create } from "@bufbuild/protobuf";

import type { Directive } from "./generated/tdl/ir/v1/ir_pb.js";
import {
	type DirectiveSpec,
	FeaturesSchema,
	type HandshakeReply,
	HandshakeReplySchema,
	type ImportRequest,
	type ImportResponse,
	type Numbering,
	Placement,
	type Request,
	type Response,
} from "./generated/tdl/plugin/v1/plugin_pb.js";

/** A backend turns a resolved model into files. */
export interface Backend {
	/**
	 * Reports what this backend is and what directives it understands. tdl
	 * checks a target block against them before generating.
	 */
	describe(): Description;

	/**
	 * Returns the files for one request. It throws only when it cannot
	 * produce a response at all; a problem with the model belongs in the
	 * response's diagnostics.
	 */
	generate(request: Request): Response | Promise<Response>;

	/**
	 * Returns the model the request's files describe, or TDL source the host
	 * lowers, for `tdl import`. A backend that imports implements it and sets
	 * {@link Description.reverse}; {@link serve} refuses an import handshake
	 * to one that does not.
	 */
	import?(request: ImportRequest): ImportResponse | Promise<ImportResponse>;
}

/** What a backend says about itself, answered in the handshake. */
export interface Description {
	/** What the backend is called, in a target block and as `tdl-gen-<name>` on PATH. */
	name: string;
	/** The backend's own version. */
	version: string;
	/** The directives the backend understands. */
	directives?: DirectiveSpec[];
	/** Says the backend can serve more than one request on a connection. */
	reuse?: boolean;
	/**
	 * Says the backend implements {@link Backend.import}. Set without it, the
	 * handshake accepts an import whose first request then fails.
	 */
	reverse?: boolean;
	/** How the backend numbers fields and variants, or unset for a backend that does not. */
	numbering?: Numbering;
	/**
	 * Says the same request may not produce the same files, so the host keeps
	 * what it wrote while the request is unchanged.
	 */
	nondeterministic?: boolean;
}

/** The handshake reply carrying a description. */
export function reply(desc: Description): HandshakeReply {
	return create(HandshakeReplySchema, {
		accepted: true,
		name: desc.name,
		version: desc.version,
		directives: desc.directives ?? [],
		features: create(FeaturesSchema, {
			reuse: desc.reuse ?? false,
			reverse: desc.reverse ?? false,
			nondeterministic: desc.nondeterministic ?? false,
		}),
		...(desc.numbering && { numbering: desc.numbering }),
	});
}

/**
 * The directives in `all` that belong to `target`. A request from tdl
 * carries only its target's directives, so this matters for a model built
 * some other way.
 */
export function directivesFor(target: string, all: readonly Directive[]): Directive[] {
	return all.filter((d) => d.target === target);
}

/** Where a name for a declaration, a field, or a variant goes. */
export const ON_MEMBERS: readonly Placement[] = [Placement.DECL, Placement.FIELD, Placement.VARIANT];
