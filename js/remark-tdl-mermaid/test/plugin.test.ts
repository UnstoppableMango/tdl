// The plugin against a real tdl and tdl-gen-mermaid: TDL names the tdl
// executable, or tdl is found on PATH, and without either the tests skip.
// checks.remark-tdl-mermaid runs them against the packaged tdl, which ships
// tdl-gen-mermaid beside it.

import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { accessSync, constants, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, dirname, join } from "node:path";
import { describe, test } from "node:test";

import type { Code, Root } from "mdast";
import { remark } from "remark";
import { VFile } from "vfile";

import remarkTdlMermaid from "../src/index.js";

function findTdl(): string | undefined {
	if (process.env.TDL) return process.env.TDL;
	for (const dir of (process.env.PATH ?? "").split(delimiter)) {
		if (!dir) continue;
		try {
			accessSync(join(dir, "tdl"), constants.X_OK);
			return join(dir, "tdl");
		} catch {}
	}
	return undefined;
}

const tdl = findTdl();
const skip = tdl === undefined ? "no tdl: set TDL or put tdl on PATH" : false;

async function run(markdown: string): Promise<{ tree: Root; file: VFile }> {
	const processor = remark().use(remarkTdlMermaid, { tdl: tdl as string });
	const file = new VFile({ path: "doc.md", value: markdown });
	const tree = (await processor.run(processor.parse(file), file)) as Root;
	return { tree, file };
}

function codes(tree: Root): Code[] {
	return tree.children.filter((n): n is Code => n.type === "code");
}

// What tdl gen writes for the same source, run directly.
function generate(source: string, id: string): string {
	const dir = mkdtempSync(join(tmpdir(), "remark-tdl-mermaid-test-"));
	try {
		writeFileSync(join(dir, "source.tdl"), source);
		execFileSync(tdl as string, ["gen", "--target", "mermaid", "-o", join(dir, "out"), join(dir, "source.tdl")], {
			stdio: ["ignore", "ignore", "pipe"],
			// As the plugin runs it, finding tdl-gen-mermaid beside tdl.
			env: { ...process.env, PATH: [dirname(tdl as string), process.env.PATH].join(delimiter) },
		});
		return readFileSync(join(dir, "out", `${id}.mmd`), "utf8");
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
}

const shop = `package shop

type Order {
  id: string
  lines: [LineItem]
}

type LineItem {
  sku: string
  quantity: int32
}
`;

describe("remark-tdl-mermaid", { skip }, () => {
	test("a mermaid fence becomes the diagram tdl gen writes", async () => {
		const { tree, file } = await run(`# Shop\n\n\`\`\`tdl mermaid\n${shop}\`\`\`\n`);
		const [code] = codes(tree);
		assert.ok(code);
		assert.equal(code.lang, "mermaid");
		assert.equal(code.meta, null);
		assert.equal(`${code.value}\n`, generate(`${shop}\ntarget mermaid for shop { }\n`, "shop"));
		assert.deepEqual(file.messages, []);
	});

	test("a plain tdl fence is left alone", async () => {
		const md = `\`\`\`tdl\n${shop}\`\`\`\n\n\`\`\`tdl other\n${shop}\`\`\`\n\n\`\`\`ts mermaid\nlet x = 1\n\`\`\`\n`;
		const { tree } = await run(md);
		assert.deepEqual(
			codes(tree).map((c) => [c.lang, c.meta, c.value]),
			[
				["tdl", null, shop.trimEnd()],
				["tdl", "other", shop.trimEnd()],
				["ts", "mermaid", "let x = 1"],
			],
		);
	});

	test("a fence's own target mermaid block is honored", async () => {
		const source = `package widgets

primitive Fn: type -> type -> type

type Widget {
  id: string
}

type WidgetService {
  getWidget: Fn<string, Widget>
}

target mermaid for widgets {
  Fn => rpc
  WidgetService => service
}
`;
		const { tree } = await run(`\`\`\`tdl mermaid\n${source}\`\`\`\n`);
		const [code] = codes(tree);
		assert.ok(code);
		assert.equal(`${code.value}\n`, generate(source, "widgets"));
		assert.match(code.value, /\+getWidget\(string\) Widget/);
	});

	test("a lowering error fails the file at its markdown line", async () => {
		const md = `# Shop\n\nSome text.\n\n\`\`\`tdl mermaid\npackage shop\n\ntype Order {\n  lines: [Nope]\n}\n\`\`\`\n`;
		await assert.rejects(run(md), (err: unknown) => {
			assert.ok(err instanceof Error);
			const message = err as Error & { line?: number; column?: number; file?: string };
			assert.match(message.message, /undefined: Nope/);
			// The fence opens on line 5, so the fence's line 4 is line 9.
			assert.equal(message.line, 9);
			assert.equal(message.column, 11);
			return true;
		});
	});

	test("a backend warning is a positioned message, and the diagram is still written", async () => {
		const md = `\`\`\`tdl mermaid\npackage shop\n\ntype Order {\n  id: string\n}\n\ntarget mermaid for shop {\n  direction(LR)\n}\n\`\`\`\n`;
		const { tree, file } = await run(md);
		assert.equal(codes(tree)[0]?.lang, "mermaid");
		assert.equal(file.messages.length, 1);
		const [warning] = file.messages;
		assert.ok(warning);
		assert.match(warning.reason, /does not declare the directive direction/);
		assert.equal(warning.fatal, false);
		assert.equal(warning.line, 9);
		assert.equal(warning.column, 3);
		assert.equal(warning.source, "tdl");
	});

	test("a fence without a package fails the file at the fence", async () => {
		const md = `Text.\n\n\`\`\`tdl mermaid\ntype Order {\n  id: string\n}\n\`\`\`\n`;
		await assert.rejects(run(md), (err: unknown) => {
			const message = err as Error & { line?: number };
			assert.match(message.message, /package/);
			assert.equal(message.line, 3);
			return true;
		});
	});

	test("a missing tdl fails the file", async () => {
		const processor = remark().use(remarkTdlMermaid, { tdl: join(tmpdir(), "no-such-tdl") });
		const file = new VFile({ value: `\`\`\`tdl mermaid\n${shop}\`\`\`\n` });
		await assert.rejects(processor.run(processor.parse(file), file), (err: unknown) => {
			const message = err as Error & { line?: number; source?: string };
			assert.match(message.message, /no-such-tdl/);
			assert.equal(message.source, "tdl");
			assert.equal(message.line, 1);
			return true;
		});
	});

	test("a commented-out target block does not count as the fence's own", async () => {
		const source = `package shop\n\n// target mermaid for shop { }\ntype Order {\n  id: string\n}\n`;
		const { tree } = await run(`\`\`\`tdl mermaid\n${source}\`\`\`\n`);
		assert.equal(codes(tree)[0]?.lang, "mermaid");
	});
});
