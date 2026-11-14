import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { camel, fromScreaming, lastSegment, pascal, screamingSnake, snake, words } from "../src/case.js";

describe("case styles", () => {
	test("join words", () => {
		const ws = ["user", "ID"];
		assert.equal(pascal(ws), "UserID");
		assert.equal(camel(ws), "userID");
		assert.equal(snake(ws), "user_id");
		assert.equal(screamingSnake(ws), "USER_ID");
		assert.equal(pascal(["user", "id"]), "UserId");
		assert.equal(camel(["HTTP", "server"]), "httpServer");
		assert.equal(camel([]), "");
		assert.equal(pascal([]), "");
	});

	test("words splits as tdl does", () => {
		const cases: [string, string[]][] = [
			["userID", ["user", "ID"]],
			["HTTPServer", ["HTTP", "Server"]],
			["last4", ["last4"]],
			["last4Digits", ["last4", "Digits"]],
			["line_items", ["line", "items"]],
			["OrderLine", ["Order", "Line"]],
			["ID", ["ID"]],
			["a-b c", ["a", "b", "c"]],
			["", []],
		];
		for (const [name, want] of cases) {
			assert.deepEqual(words(name), want, name);
		}
	});

	test("lastSegment is after the last dot", () => {
		assert.equal(lastSegment("shop.billing.Invoice"), "Invoice");
		assert.equal(lastSegment("Invoice"), "Invoice");
	});

	test("fromScreaming is pascal", () => {
		assert.equal(fromScreaming("IN_TRANSIT"), "InTransit");
		assert.equal(fromScreaming("_A__B_"), "AB");
	});
});
