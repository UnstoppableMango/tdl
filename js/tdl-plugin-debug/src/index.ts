/**
 * A port of the Go debug backend, which describes the model it was given. It
 * exercises the SDK and the protocol without generating code, and answers a
 * request with the bytes the Go backend answers it with.
 *
 * @module
 */

import { create } from "@bufbuild/protobuf";
import {
	type Backend,
	type Description,
	directivesFor,
	fields,
	ir,
	protocol,
	Session,
	unsupported,
} from "@unmango/tdl-plugin";

/**
 * What this backend is called, in a target block and as `tdl-gen-js_debug`
 * on PATH. The Go backend holds the name `debug`, and tdl never looks a
 * built-in name up on PATH.
 */
export const NAME = "js_debug";

/** The debug backend. */
export const debug: Backend = {
	describe(): Description {
		return {
			name: NAME,
			version: "0.1.0",
			// Requests share no state.
			reuse: true,
			directives: [
				// note("...") exists to exercise directive plumbing.
				create(protocol.DirectiveSpecSchema, {
					name: "note",
					minArgs: 1,
					maxArgs: 1,
					argKinds: [ir.LiteralKind.STRING],
				}),
			],
		};
	},

	/** Writes one file describing the model. */
	generate(request: protocol.Request): protocol.Response {
		const s = new Session(request, "debug");
		const model = s.model();
		const decls = s.decls();
		const own = s.own();

		let b =
			`package ${model?.package ?? ""}\ntarget ${request.target}\n\n` +
			`declarations: ${own.length} own, ${decls.length - own.length} from the prelude\n` +
			`types: ${model?.types.length ?? 0}\nexterns: ${model?.externs.length ?? 0}\n` +
			`instances: ${model?.instances.length ?? 0}\n`;

		for (const d of own) {
			b += `\n${d.meta?.name ?? ""}\n`;
			for (const dir of directivesFor(request.target, d.directives)) {
				b += `  directive ${dir.name}\n`;
			}
			for (const f of fields(d)) {
				b += `  field ${f.meta?.name ?? ""}: ${f.type?.name ?? ""}\n`;
			}
		}

		// Each fieldless declaration warns, to exercise the diagnostic path.
		for (const d of own) {
			if (fields(d).length > 0 || d.node.case === "enumeration") {
				continue;
			}
			s.warn(
				unsupported(`${d.meta?.name ?? ""} has no fields, so there is nothing to generate for it`, d.meta?.position),
			);
		}

		return s.response([create(protocol.FileSchema, { path: "model.txt", content: new TextEncoder().encode(b) })]);
	},
};
