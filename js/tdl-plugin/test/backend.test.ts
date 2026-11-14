import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { create } from "@bufbuild/protobuf";

import { directivesFor, reply } from "../src/backend.js";
import { DirectiveSchema } from "../src/generated/tdl/ir/v1/ir_pb.js";
import { DirectiveSpecSchema } from "../src/generated/tdl/plugin/v1/plugin_pb.js";

describe("reply", () => {
	test("carries the description", () => {
		const directives = [create(DirectiveSpecSchema, { name: "note" })];

		const got = reply({ name: "mine", version: "1.2.3", reuse: true, nondeterministic: true, directives });

		assert.ok(got.accepted);
		assert.equal(got.name, "mine");
		assert.equal(got.version, "1.2.3");
		assert.deepEqual(got.directives, directives);
		assert.equal(got.features?.reuse, true);
		assert.equal(got.features?.reverse, false);
		assert.equal(got.features?.nondeterministic, true);
		assert.equal(got.numbering, undefined);
		assert.equal(reply({ name: "mine", version: "1.2.3" }).features?.nondeterministic, false);
	});
});

describe("directivesFor", () => {
	test("keeps the target's", () => {
		const all = [
			create(DirectiveSchema, { name: "a", target: "go" }),
			create(DirectiveSchema, { name: "b", target: "js" }),
		];

		assert.deepEqual(
			directivesFor("js", all).map((d) => d.name),
			["b"],
		);
	});
});
