import { create } from "@bufbuild/protobuf";

import { directivesFor } from "./backend.js";
import { lastSegment, type Style, words as splitWords } from "./case.js";
import { ProtocolError } from "./error.js";
import type {
	Constraint,
	Decl,
	Directive,
	Field,
	ID,
	Meta,
	Model,
	Param,
	Position,
	Variant,
} from "./generated/tdl/ir/v1/ir_pb.js";
import {
	type Analysis,
	type ConstraintView,
	type DeclView,
	type Diagnostic,
	DiagnosticSchema,
	type File,
	type MemberView,
	type Request,
	type Response,
	ResponseSchema,
	Severity,
	type TypeView,
} from "./generated/tdl/plugin/v1/plugin_pb.js";
import { type LossCode, loss, type Problem, unsupported } from "./problem.js";

/**
 * The declarations a backend does not generate, by their identity in the
 * request.
 */
export class Skipped {
	readonly #decls = new Set<Decl>();

	/** Adds a declaration, and says whether it was not already there. */
	add(d: Decl): boolean {
		const had = this.#decls.has(d);
		this.#decls.add(d);
		return !had;
	}

	/** Whether a declaration is in the set. */
	has(d: Decl): boolean {
		return this.#decls.has(d);
	}

	/** How many declarations are in the set. */
	get size(): number {
		return this.#decls.size;
	}
}

/**
 * One request's state: the model, the analysis tdl sent with it, and the
 * diagnostics the backend collects. A backend makes a fresh one per request.
 *
 * The model is reached through methods only. The views are looked up by the
 * node's identity in the request, so a node cloned out of it has none.
 */
export class Session {
	readonly #request: Request;
	readonly #analysis: Analysis;
	readonly #lang: string;
	readonly #decls = new Map<Decl, number>();
	readonly #members = new Map<Field | Variant, MemberView>();
	readonly #constraints = new Map<Constraint, ConstraintView>();

	/** Makes {@link lose} report nothing, since annotations carry what it would. */
	roundtrip = false;

	/**
	 * Makes {@link lose} report read-back loss, which it otherwise leaves
	 * out: a backend sets it from a bare `readback` directive in its target
	 * block.
	 */
	readback = false;

	/** The diagnostics collected so far. */
	readonly diagnostics: Diagnostic[] = [];

	/**
	 * Starts a session for one request. `lang` is the target language as a
	 * message names it. It throws a {@link ProtocolError} when the request
	 * carries no analysis for its model, which tdl sends with every request.
	 */
	constructor(request: Request, lang: string) {
		const analysis = request.analysis;
		if (analysis === undefined) {
			throw new ProtocolError("the request carries no analysis; tdl sends one with every request");
		}
		const model = request.model;
		const decls = model?.decls ?? [];
		const ok =
			analysis.decls.length === decls.length &&
			analysis.types.length === (model?.types.length ?? 0) &&
			decls.every((d, i) => describes(d, analysis.decls[i] as DeclView));
		if (!ok) {
			throw new ProtocolError("the request's analysis does not describe its model");
		}

		this.#request = request;
		this.#analysis = analysis;
		this.#lang = lang;
		decls.forEach((d, i) => {
			const view = analysis.decls[i] as DeclView;
			this.#decls.set(d, i);
			if (d.node.case === "newtype") {
				this.#indexConstraints(d.node.value.valueConstraints, view.constraints);
			}
			this.#indexFields(fields(d), view.fields);
			variants(d).forEach((v, j) => {
				const vv = view.variants[j] as MemberView;
				this.#members.set(v, vv);
				this.#indexFields(v.fields, vv.fields);
			});
		});
	}

	#indexFields(fs: readonly Field[], views: readonly MemberView[]): void {
		fs.forEach((f, i) => {
			const view = views[i] as MemberView;
			this.#members.set(f, view);
			this.#indexConstraints(f.constraints, view.constraints);
		});
	}

	#indexConstraints(cs: readonly Constraint[], views: readonly ConstraintView[]): void {
		cs.forEach((c, i) => {
			this.#constraints.set(c, views[i] as ConstraintView);
		});
	}

	/** The request being served. */
	request(): Request {
		return this.#request;
	}

	/** The model, with the prelude's declarations merged into it. */
	model(): Model | undefined {
		return this.#request.model;
	}

	/** The model's declarations, the prelude's included. */
	decls(): readonly Decl[] {
		return this.#request.model?.decls ?? [];
	}

	/** The target block being served. */
	target(): string {
		return this.#request.target;
	}

	/** The target language as a message names it. */
	lang(): string {
		return this.#lang;
	}

	/** What tdl derived from the model for this target. */
	analysis(): Analysis {
		return this.#analysis;
	}

	/** The response holding files and every diagnostic collected. */
	response(files: File[]): Response {
		return create(ResponseSchema, { files, diagnostics: this.diagnostics });
	}

	/** The analysis of a declaration in the model, or undefined for one not in it. */
	declView(d: Decl): DeclView | undefined {
		const i = this.#decls.get(d);
		return i === undefined ? undefined : this.#analysis.decls[i];
	}

	/** The analysis of a field of a declaration in the model, or undefined. */
	fieldView(f: Field): MemberView | undefined {
		return this.#members.get(f);
	}

	/** The analysis of an enum's variant in the model, or undefined. */
	variantView(v: Variant): MemberView | undefined {
		return this.#members.get(v);
	}

	/** The analysis of a constraint on a newtype or a field in the model, or undefined. */
	constraintView(c: Constraint): ConstraintView | undefined {
		return this.#constraints.get(c);
	}

	/** A constraint as TDL source writes it, as `min(0)`. */
	source(c: Constraint): string {
		return this.constraintView(c)?.source ?? c.name;
	}

	/** The analysis of a type reference, or undefined when it does not resolve. */
	typeView(id: ID): TypeView | undefined {
		return this.#analysis.types[id.index];
	}

	/** The declaration an ID names, or undefined when it does not resolve. */
	declAt(id: ID): Decl | undefined {
		return this.decls()[id.index];
	}

	/** The declarations that did not come from the embedded prelude. */
	own(): Decl[] {
		return this.decls().filter((_, i) => this.#analysis.decls[i]?.own);
	}

	/** The session target's directive with this name carrying at least one argument, or undefined. */
	find(all: readonly Directive[], name: string): Directive | undefined {
		return directivesFor(this.target(), all).find((d) => d.name === name && d.args.length > 0);
	}

	/** The first argument of a directive on a node, as written, or undefined. */
	text(all: readonly Directive[], name: string): string | undefined {
		return this.find(all, name)?.args[0]?.text;
	}

	/**
	 * A directive written on the target block itself, returned whole so a
	 * diagnostic about its value has its position.
	 */
	block(name: string): Directive | undefined {
		for (const b of this.#blocks()) {
			const d = this.find(b.directives, name);
			if (d) {
				return d;
			}
		}
		return undefined;
	}

	/** Whether the session target carries a directive with this name among `all`, with or without arguments. */
	tagged(all: readonly Directive[], name: string): boolean {
		return directivesFor(this.target(), all).some((d) => d.name === name);
	}

	/** Whether this target's block carries a directive with this name, such as a bare `readback`. */
	bare(name: string): boolean {
		return this.#blocks().some((b) => this.tagged(b.directives, name));
	}

	#blocks() {
		return (this.#request.model?.targets ?? []).filter((b) => b.meta?.name === this.target());
	}

	/**
	 * A declaration's name in the target language: a `name` directive's
	 * argument when there is one, otherwise its words joined by style.
	 */
	declName(d: Decl, style: Style): string {
		return this.#name(d.directives, this.declView(d)?.words, lastSegment(d.meta?.name ?? ""), style);
	}

	/** A field's name in the target language, by the rule {@link declName} follows. */
	fieldName(f: Field, style: Style): string {
		return this.#name(f.directives, this.fieldView(f)?.words, f.meta?.name ?? "", style);
	}

	/** A variant's name in the target language, by the rule {@link declName} follows. */
	variantName(v: Variant, style: Style): string {
		return this.#name(v.directives, this.variantView(v)?.words, v.meta?.name ?? "", style);
	}

	// The rule the three names share: a name directive, else the shipped
	// words, else the words of the name as written, for a node the request
	// does not hold.
	#name(directives: readonly Directive[], words: readonly string[] | undefined, written: string, style: Style) {
		return this.text(directives, "name") ?? style(words ?? splitWords(written));
	}

	/**
	 * Reports a problem: an unsupported one as a warning with its loss code,
	 * and an invalid one as an error.
	 */
	warn(problem: Problem): void {
		if (problem.kind === "unsupported") {
			this.diagnostics.push(diagnostic(Severity.WARNING, problem.message, problem.position, problem.code));
		} else {
			this.diagnostics.push(diagnostic(Severity.ERROR, problem.message, problem.position, ""));
		}
	}

	/** Reports a problem that makes the output unusable. The host writes nothing when a response carries one. */
	error(position: Position | undefined, message: string): void {
		this.diagnostics.push(diagnostic(Severity.ERROR, message, position, ""));
	}

	/**
	 * Warns that a fact is lost, with the loss code that names it. It reports
	 * output loss: what the output itself drops or weakens, which annotations
	 * do not restore, so it warns under roundtrip too. {@link lose} reports
	 * read-back loss.
	 */
	lossy(code: LossCode, position: Position | undefined, message: string): void {
		this.diagnostics.push(diagnostic(Severity.WARNING, message, position, code));
	}

	/**
	 * Warns that a fact the output keeps is one reading it back would not
	 * recover. It reports only under {@link readback}, and never under
	 * {@link roundtrip}, where an annotation carries the fact. {@link lossy}
	 * reports output loss.
	 */
	lose(code: LossCode, position: Position | undefined, message: string): void {
		if (this.readback && !this.roundtrip) {
			this.lossy(code, position, message);
		}
	}

	/**
	 * Reports a declaration that is not generated. It is output loss, so it
	 * warns under roundtrip too, where the declaration is carried as TDL but
	 * the output still lacks it.
	 */
	skip(problem: Problem): void {
		this.warn(problem);
	}

	/**
	 * Warns that a declaration's constraints are not enforced: a newtype's
	 * where block and each field's, including enum variants'. `enforced`
	 * reports the constraints the target's types enforce themselves, which do
	 * not warn.
	 */
	warnConstraints(d: Decl, enforced: (c: Constraint) => boolean): void {
		const name = d.meta?.name ?? "";
		const unenforced = (cs: readonly Constraint[]) => cs.filter((c) => !enforced(c)).length;
		if (d.node.case === "newtype") {
			const n = unenforced(d.node.value.valueConstraints);
			if (n > 0) {
				this.lossy(
					loss.CONSTRAINT,
					d.meta?.position,
					`${name} carries ${n} where constraint(s), and validation is not generated yet`,
				);
			}
		}

		for (const f of [...fields(d), ...variants(d).flatMap((v) => v.fields)]) {
			const m = unenforced(f.constraints);
			if (m > 0) {
				this.lossy(
					loss.CONSTRAINT,
					f.meta?.position,
					`${lastSegment(name)}.${f.meta?.name ?? ""} carries ${m} constraint(s), and validation is not generated yet`,
				);
			}
		}
	}

	/**
	 * The declarations `d` names through its fields, its variants' fields, a
	 * newtype's base, and their type arguments, with aliases expanded. A
	 * declaration naming itself is included.
	 */
	references(d: Decl): Decl[] {
		return this.#declsNamed(this.declView(d)?.references ?? []);
	}

	/** The declarations the type references name, walked as {@link references} walks a declaration's, each once. */
	typeReferences(types: readonly ID[]): Decl[] {
		const seen = new Set<number>();
		const refs = types
			.flatMap((t) => this.typeView(t)?.references ?? [])
			.filter((r) => !seen.has(r.index) && seen.add(r.index));
		return this.#declsNamed(refs);
	}

	#declsNamed(ids: readonly ID[]): Decl[] {
		return ids.map((id) => this.declAt(id)).filter((d) => d !== undefined);
	}

	/**
	 * Extends `skipped`, transitively, with every declaration in `decls` that
	 * names a skipped one, and warns about each it adds, so the output never
	 * refers to something it does not declare.
	 */
	cascade(decls: readonly Decl[], skipped: Skipped): void {
		let changed = true;
		while (changed) {
			changed = false;
			for (const d of decls) {
				// An alias emits nothing; references already expands it.
				if (skipped.has(d) || d.node.case === "alias") {
					continue;
				}
				const named = this.references(d).find((r) => skipped.has(r));
				if (named === undefined) {
					continue;
				}
				skipped.add(d);
				changed = true;
				this.warn(
					unsupported(
						`${d.meta?.name ?? ""} names ${named.meta?.name ?? ""}, which is not generated`,
						d.meta?.position,
					),
				);
			}
		}
	}
}

// Whether a declaration's view has one view per field, variant, and
// constraint, as tdl's analysis writes it, so no node goes without its view.
function describes(d: Decl, view: DeclView): boolean {
	const fieldsDescribed = (fs: readonly Field[], views: readonly MemberView[]) =>
		fs.length === views.length && fs.every((f, i) => f.constraints.length === views[i]?.constraints.length);
	const constraints = d.node.case === "newtype" ? d.node.value.valueConstraints.length : 0;
	const vs = variants(d);
	return (
		constraints === view.constraints.length &&
		fieldsDescribed(fields(d), view.fields) &&
		vs.length === view.variants.length &&
		vs.every((v, i) => fieldsDescribed(v.fields, (view.variants[i] as MemberView).fields))
	);
}

function diagnostic(severity: Severity, message: string, position: Position | undefined, code: string): Diagnostic {
	return create(DiagnosticSchema, { severity, message, code, ...(position && { position }) });
}

/** A struct declaration's fields; an enum's belong to its variants. */
export function fields(d: Decl): readonly Field[] {
	return d.node.case === "structure" ? d.node.value.fields : [];
}

/** An enum declaration's variants. */
export function variants(d: Decl): readonly Variant[] {
	return d.node.case === "enumeration" ? d.node.value.variants : [];
}

/** The type parameters of a declaration. */
export function params(d: Decl): readonly Param[] {
	switch (d.node.case) {
		case "alias":
		case "newtype":
		case "structure":
		case "enumeration":
		case "class":
			return d.node.value.params;
		default:
			return [];
	}
}

/**
 * Whether a schema backend writes anything for a declaration, or the problem
 * that keeps it from generating one: an alias is expanded where it is used
 * and a primitive names an opaque type, so neither declares anything, while
 * classes, units, and type parameters are not generated yet.
 */
export function declares(d: Decl): boolean | Problem {
	const pos = d.meta?.position;
	const name = d.meta?.name ?? "";
	switch (d.node.case) {
		case "class":
			return unsupported(`${name} is a class, and classes are not generated yet`, pos);
		case "unit":
			return unsupported(`${name} is a unit, and units are not generated yet`, pos);
		case "structure":
		case "enumeration":
		case "newtype":
			return params(d).length === 0
				? true
				: unsupported(`${name} is parameterized, and generics are not generated yet`, pos);
		default:
			return false;
	}
}

/** A node's documentation, one line per entry, with trailing whitespace removed and indentation kept. */
export function doc(m: Meta | undefined): string[] {
	return (m?.doc ?? []).map((l) => l.replace(/[ \t]+$/, ""));
}

/** Whether a node is deprecated: the reason, which may be empty, or undefined when it is not. */
export function deprecated(m: Meta | undefined): string | undefined {
	return m?.deprecated?.reason;
}
