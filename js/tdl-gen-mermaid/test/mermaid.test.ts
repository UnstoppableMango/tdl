// The backend answers each request recorded in testdata/mermaid with the
// files beside it, and Mermaid's own parser accepts every diagram.
//
// Go records the requests (go test ./internal/gen -record), since these
// tests cannot lower TDL. TDL_RECORD=1 rewrites the files they compare.

import assert from "node:assert/strict";
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { before, describe, test } from "node:test";
import { fileURLToPath } from "node:url";

import { fromBinary } from "@bufbuild/protobuf";
import { protocol as pb, type protocol } from "@unmango/tdl-plugin";
import { JSDOM } from "jsdom";

import { HEADER, mermaid, packageId } from "../src/index.js";

const record = process.env.TDL_RECORD === "1";

function testdata(): string {
	let dir = dirname(fileURLToPath(import.meta.url));
	while (!existsSync(join(dir, "testdata", "mermaid"))) {
		const up = dirname(dir);
		assert.notEqual(up, dir, "no testdata/mermaid above the tests");
		dir = up;
	}
	return join(dir, "testdata", "mermaid");
}

// Each recorded request with the directory its files belong in: a case's
// own want/ beside its source, and a corpus case's want/.
function recorded(): { name: string; request: string; want: string }[] {
	const root = testdata();
	const out = [];
	for (const c of readdirSync(join(root, "cases")).sort()) {
		for (const f of readdirSync(join(root, "cases", c)).sort()) {
			if (f.endsWith(".request.binpb")) {
				const base = f.slice(0, -".request.binpb".length);
				const want = base === "source" ? "want" : `want-${base}`;
				out.push({
					name: `cases/${c}/${base}`,
					request: join(root, "cases", c, f),
					want: join(root, "cases", c, want),
				});
			}
		}
	}
	for (const c of readdirSync(join(root, "corpus")).sort()) {
		out.push({
			name: `corpus/${c}`,
			request: join(root, "corpus", c, "request.binpb"),
			want: join(root, "corpus", c, "want"),
		});
	}
	return out;
}

const read = (path: string) => fromBinary(pb.RequestSchema, readFileSync(path));
const text = (f: protocol.File) => new TextDecoder().decode(f.content);

// The diagram a file holds: all of a .mmd, or a .md's fenced block.
function diagram(f: protocol.File): string {
	const t = text(f);
	if (!f.path.endsWith(".md")) {
		return t;
	}
	const m = /^```mermaid\n([\s\S]*)```\n$/.exec(t);
	assert.ok(m, `${f.path} holds one fenced mermaid block`);
	return m[1] as string;
}

// mermaid.parse, which needs a DOM: jsdom's window, installed as a global.
let parse: (text: string) => Promise<unknown>;

before(async () => {
	const dom = new JSDOM("<!doctype html><html><body></body></html>", { pretendToBeVisual: true });
	Object.assign(globalThis, { window: dom.window, document: dom.window.document });
	const { default: m } = await import("mermaid");
	parse = (t) => m.parse(t);
});

describe("mermaid", () => {
	test("describes itself", () => {
		const d = mermaid.describe();

		assert.equal(d.name, "mermaid");
		assert.ok(d.reuse);
		assert.deepEqual(
			d.directives?.map((s) => s.name),
			["format", "service", "rpc", "stream"],
		);
	});

	test("names the file for the package", () => {
		assert.equal(packageId("shop.billing"), "shop_billing");
		assert.equal(packageId(""), "model");
	});

	for (const c of recorded()) {
		test(`draws ${c.name}`, async () => {
			const resp = await mermaid.generate(read(c.request));

			assert.deepEqual(
				resp.diagnostics.filter((d) => d.severity === pb.Severity.ERROR),
				[],
			);
			for (const f of resp.files) {
				assert.ok(diagram(f).startsWith(`${HEADER}\nclassDiagram\n`), f.path);
				await parse(diagram(f));
			}
			checkFiles(c.want, resp.files);
		});
	}
});

function checkFiles(dir: string, files: protocol.File[]) {
	if (record) {
		rmSync(dir, { recursive: true, force: true });
		for (const f of files) {
			mkdirSync(dirname(join(dir, f.path)), { recursive: true });
			writeFileSync(join(dir, f.path), f.content);
		}
		return;
	}
	assert.deepEqual(files.map((f) => f.path).sort(), existsSync(dir) ? readdirSync(dir).sort() : []);
	for (const f of files) {
		assert.equal(text(f), readFileSync(join(dir, f.path), "utf8"), `${f.path} (rerun with TDL_RECORD=1)`);
	}
}
