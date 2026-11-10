import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { clone, create } from "@bufbuild/protobuf";

import { camel, screamingSnake, snake } from "../src/case.js";
import { ProtocolError } from "../src/error.js";
import {
	type Decl,
	DeclSchema,
	DirectiveSchema,
	IDSchema,
	LiteralSchema,
	MetaSchema,
	ModelSchema,
	PositionSchema,
	StructSchema,
	TypeSchema,
} from "../src/generated/tdl/ir/v1/ir_pb.js";
import {
	AnalysisSchema,
	DeclViewSchema,
	type Request,
	RequestSchema,
	Severity,
	TypeViewSchema,
} from "../src/generated/tdl/plugin/v1/plugin_pb.js";
import { invalid, loss, lossy, unsupported } from "../src/problem.js";
import { declares, fields, Session, Skipped, variants } from "../src/session.js";
import { request } from "./helpers.js";

function named(s: Session, name: string): Decl {
	const d = s.decls().find((d) => d.meta?.name === name);
	assert.ok(d, name);
	return d;
}

describe("Session", () => {
	test("refuses a request without an analysis", () => {
		const req = request("shop");
		req.analysis = undefined;

		assert.throws(
			() => new Session(req, "Debug"),
			(e: unknown) => {
				assert.ok(e instanceof ProtocolError);
				assert.match(e.message, /carries no analysis/);
				return true;
			},
		);
	});

	test("refuses an analysis of another model", () => {
		const req = request("shop");
		req.analysis?.decls.shift();

		assert.throws(() => new Session(req, "Debug"), ProtocolError);
	});

	test("refuses an analysis missing a field's view", () => {
		const req = request("catalog");
		const i = req.model?.decls.findIndex((d) => d.meta?.name === "Product") ?? -1;
		req.analysis?.decls[i]?.fields.pop();

		assert.throws(() => new Session(req, "Debug"), ProtocolError);
	});

	test("own leaves out the prelude", () => {
		const s = new Session(request("catalog"), "Debug");

		assert.deepEqual(
			s.own().map((d) => d.meta?.name),
			["Product", "Marker", "Availability"],
		);
	});

	test("names from the shipped words", () => {
		const s = new Session(request("catalog"), "Debug");
		const product = named(s, "Product");
		const id = fields(product)[0];
		const inStock = variants(named(s, "Availability"))[0];
		assert.ok(id && inStock);

		assert.equal(s.declName(product, snake), "product");
		assert.equal(s.fieldName(id, screamingSnake), "ID");
		assert.equal(s.variantName(inStock, snake), "in_stock");
		assert.equal(s.variantName(inStock, camel), "inStock");
	});

	test("finds a view by identity", () => {
		const s = new Session(request("catalog"), "Debug");
		const product = named(s, "Product");
		const copy = clone(DeclSchema, product);

		assert.equal(s.declView(product)?.own, true);
		assert.equal(s.declView(copy), undefined);
		// The name falls back to splitting the declaration's own name.
		assert.equal(s.declName(copy, snake), "product");
		const name = fields(product)[1];
		assert.ok(name?.constraints[0]);
		assert.equal(s.source(name.constraints[0]), "length(1..64)");
	});

	test("a name directive wins", () => {
		const req = request("catalog");
		const product = req.model?.decls.find((d) => d.meta?.name === "Product");
		for (const [target, text] of [
			["debug", "Item"],
			["other", "Other"],
		] as const) {
			product?.directives.push(
				create(DirectiveSchema, { name: "name", target, args: [create(LiteralSchema, { text })] }),
			);
		}
		const s = new Session(req, "Debug");

		assert.equal(s.declName(named(s, "Product"), snake), "Item");
	});

	test("reads the target block", () => {
		const s = new Session(request("catalog"), "Debug");

		assert.equal(s.block("out")?.args[0]?.text, "./gen");
		assert.equal(s.block("note"), undefined);
		assert.ok(s.bare("roundtrip"));
		assert.ok(!s.bare("closed"));
	});

	test("warns with a position and a code", () => {
		const s = new Session(request("catalog"), "Debug");
		const pos = create(PositionSchema, { filename: "a.tdl", line: 1 });

		s.warn(lossy(loss.COLLECTION, "lost", pos));
		s.warn(invalid("wrong", pos));
		s.error(undefined, "broken");
		s.lossy(loss.DOC, pos, "undocumented");

		assert.deepEqual(
			s.diagnostics.map((d) => [d.severity, d.message, d.code]),
			[
				[Severity.WARNING, "lost", loss.COLLECTION],
				[Severity.ERROR, "wrong", ""],
				[Severity.ERROR, "broken", ""],
				[Severity.WARNING, "undocumented", loss.DOC],
			],
		);
		assert.equal(s.diagnostics[0]?.position, pos);
		assert.equal(s.diagnostics[2]?.position, undefined);
		const resp = s.response([]);
		assert.equal(resp.diagnostics.filter((d) => d.severity === Severity.ERROR).length, 2);
	});

	test("output loss always warns, and read-back loss only under readback", () => {
		const cases: [boolean, boolean, string[]][] = [
			[false, false, ["output", "skipped"]],
			[true, false, ["output", "skipped", "read back"]],
			[false, true, ["output", "skipped"]],
			[true, true, ["output", "skipped"]],
		];
		for (const [readback, roundtrip, want] of cases) {
			const s = new Session(request("catalog"), "Debug");
			s.readback = readback;
			s.roundtrip = roundtrip;

			s.lossy(loss.CONSTRAINT, undefined, "output");
			s.skip(unsupported("skipped"));
			s.lose(loss.DOC, undefined, "read back");

			assert.deepEqual(
				s.diagnostics.map((d) => d.message),
				want,
				`readback ${readback}, roundtrip ${roundtrip}`,
			);
		}
	});

	test("warns about unenforced constraints", () => {
		const s = new Session(request("catalog"), "Debug");

		s.warnConstraints(named(s, "Product"), () => false);
		s.warnConstraints(named(s, "Marker"), () => false);
		s.warnConstraints(named(s, "Product"), (c) => c.name === "length");

		assert.equal(s.diagnostics.length, 1);
		assert.equal(
			s.diagnostics[0]?.message,
			"Product.name carries 1 constraint(s), and validation is not generated yet",
		);
		assert.equal(s.diagnostics[0]?.code, loss.CONSTRAINT);
	});

	test("says what a declaration declares", () => {
		const s = new Session(request("catalog"), "Debug");
		const entity = named(s, "Entity");

		assert.equal(declares(named(s, "Product")), true);
		assert.equal(declares(named(s, "string")), false);
		assert.deepEqual(
			declares(entity),
			unsupported("Entity is a class, and classes are not generated yet", entity.meta?.position),
		);
		assert.equal((declares(named(s, "Option")) as { kind: string }).kind, "unsupported");
	});
});

// A request where A names B names C, and D names nothing, with the analysis
// tdl would send.
function cascading(): Request {
	const decl = (name: string) =>
		create(DeclSchema, {
			meta: create(MetaSchema, { name }),
			node: { case: "structure", value: create(StructSchema) },
		});
	const id = (index: number, name = "") => create(IDSchema, { index, name });
	const view = (references: ReturnType<typeof id>[]) => create(DeclViewSchema, { own: true, references });
	return create(RequestSchema, {
		target: "x",
		model: create(ModelSchema, {
			decls: [decl("A"), decl("B"), decl("C"), decl("D")],
			types: [create(TypeSchema), create(TypeSchema)],
		}),
		analysis: create(AnalysisSchema, {
			types: [
				create(TypeViewSchema, { references: [id(1, "B")] }),
				create(TypeViewSchema, { references: [id(2, "C")] }),
			],
			decls: [view([id(1, "B")]), view([id(2, "C")]), view([]), view([])],
		}),
	});
}

const names = (decls: readonly Decl[]) => decls.map((d) => d.meta?.name);

describe("the cascade", () => {
	test("follows references", () => {
		const s = new Session(cascading(), "X");
		const [a, b, , d] = s.decls() as Decl[];
		const id = (index: number) => create(IDSchema, { index });

		assert.deepEqual(names(s.references(a as Decl)), ["B"]);
		assert.deepEqual(names(s.references(b as Decl)), ["C"]);
		assert.deepEqual(s.references(d as Decl), []);
		assert.deepEqual(names(s.typeReferences([id(0), id(1), id(0), id(9)])), ["B", "C"]);
	});

	test("cascades skips", () => {
		const s = new Session(cascading(), "X");
		const decls = s.decls();
		const [a, b, c, d] = decls as Decl[];
		const skipped = new Skipped();
		skipped.add(c as Decl);

		s.cascade(decls, skipped);

		assert.equal(skipped.size, 3);
		assert.ok(skipped.has(a as Decl) && skipped.has(b as Decl));
		assert.ok(!skipped.has(d as Decl));
		assert.deepEqual(
			s.diagnostics.map((d) => d.message),
			["B names C, which is not generated", "A names B, which is not generated"],
		);
	});
});
