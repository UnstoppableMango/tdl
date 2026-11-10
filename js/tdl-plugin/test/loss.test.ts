import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { test } from "node:test";

import { LOSS_CODES } from "../src/problem.js";
import { root } from "./helpers.js";

// The loss codes are the ones docs/architecture/reverse.md tabulates, in its
// order, as emit.LossCodes is in Go.
test("loss codes match the design", () => {
	const doc = readFileSync(join(root(), "docs", "architecture", "reverse.md"), "utf8");

	const documented = doc
		.split("\n")
		.map((l) => /^\| `(lossy\.[^`]*)` \|/.exec(l)?.[1])
		.filter((code) => code !== undefined);

	assert.deepEqual(documented, LOSS_CODES);
});
