// A remark plugin that draws each ```tdl mermaid fence as the Mermaid class
// diagram tdl gen writes for it. Each fence is a self-contained TDL file:
// the plugin writes it to a temporary directory, appends an empty target
// mermaid block when the fence has none, runs tdl gen --target mermaid, and
// puts the diagram in place of the fence as a ```mermaid fence.

import { execFile } from "node:child_process";
import { mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { delimiter, dirname, join } from "node:path";

import type { Code, Nodes, Parents, Root } from "mdast";
import type { VFile } from "vfile";

export interface Options {
	/** The tdl executable. By default, the one @unmango/tdl installed. */
	tdl?: string;
}

// The name the fence is written under, which tdl's diagnostics name.
const FENCE = "fence.tdl";

// One diagnostic line tdl prints: a lowering error has no severity, and a
// backend's names one and may end with a loss code.
const DIAGNOSTIC = /^fence\.tdl:(\d+):(\d+): (?:(error|warning): )?(.*?)(?: \(([a-z]+(?:\.[a-z-]+)+)\))?$/;

interface Diagnostic {
	line: number;
	column: number;
	warning: boolean;
	message: string;
	code: string | undefined;
}

/** Where a message points: a point, the fence's span, or nowhere. */
type Place = Code["position"] | { line: number; column: number };

/** A fence to draw, with where it sits in the tree. */
interface Fence {
	parent: Parents;
	index: number;
	node: Code;
}

/** The fences whose info string is tdl with the meta word mermaid. */
function fences(tree: Root): Fence[] {
	const found: Fence[] = [];
	const walk = (parent: Parents) => {
		parent.children.forEach((child: Nodes, index: number) => {
			if (child.type === "code" && child.lang === "tdl" && (child.meta ?? "").split(/\s+/).includes("mermaid")) {
				found.push({ parent, index, node: child });
			} else if ("children" in child) {
				walk(child);
			}
		});
	};
	walk(tree);
	return found;
}

/** The fence's lines that are not comments, for the two checks below. */
function code(source: string): string {
	return source
		.split("\n")
		.filter((line) => !line.trimStart().startsWith("//"))
		.join("\n");
}

/** The package a fence declares, read from its first package clause. */
function packageOf(source: string): string | undefined {
	return /^[ \t]*package[ \t]+([\w.]+)/m.exec(code(source))?.[1];
}

/** The fence's source with an empty target mermaid block, unless it has one. */
function withTarget(source: string, pkg: string): string {
	if (/^[ \t]*target[ \t]+mermaid\b/m.test(code(source))) return source;
	return `${source}${source.endsWith("\n") || source === "" ? "" : "\n"}\ntarget mermaid for ${pkg} { }\n`;
}

/** Each diagnostic line in tdl's standard error. */
function diagnostics(stderr: string): Diagnostic[] {
	const out: Diagnostic[] = [];
	for (const line of stderr.split("\n")) {
		const m = DIAGNOSTIC.exec(line);
		if (!m) continue;
		out.push({
			line: Number(m[1]),
			column: Number(m[2]),
			warning: m[3] === "warning",
			message: m[4] ?? "",
			code: m[5],
		});
	}
	return out;
}

function defaultTdl(): string {
	const { binaryPath } = createRequire(import.meta.url)("@unmango/tdl/resolve") as {
		binaryPath(name: string): string;
	};
	return binaryPath("tdl");
}

// What running tdl gave: its exit status and standard error, or why it did
// not run.
type Run = { ok: boolean; stderr: string } | { failed: string };

function run(tdl: string, args: string[], cwd: string): Promise<Run> {
	// tdl finds tdl-gen-mermaid on PATH; the platform package and Nix's tdl
	// both put it beside tdl.
	const env = { ...process.env, PATH: [dirname(tdl), process.env.PATH].filter(Boolean).join(delimiter) };
	return new Promise((resolve) => {
		execFile(tdl, args, { cwd, env, encoding: "utf8" }, (err, _stdout, stderr) => {
			if (err && typeof err.code !== "number") resolve({ failed: `running ${tdl}: ${err.message}` });
			else resolve({ ok: !err, stderr });
		});
	});
}

/** Draws each ```tdl mermaid fence as a Mermaid class diagram. */
export default function remarkTdlMermaid(options: Options = {}): (tree: Root, file: VFile) => Promise<void> {
	return async (tree: Root, file: VFile) => {
		const found = fences(tree);
		if (found.length === 0) return;

		let tdl = options.tdl;
		if (tdl === undefined) {
			try {
				tdl = defaultTdl();
			} catch (err) {
				file.fail(err instanceof Error ? err.message : String(err), { source: "tdl" });
			}
		}

		for (const { parent, index, node } of found) {
			const lines = node.value.split("\n").length;
			const report = (reason: string, place: Place) =>
				file.message(reason, { ...(place ? { place } : {}), source: "tdl" });
			function fail(reason: string, place: Place = node.position): never {
				return file.fail(reason, { ...(place ? { place } : {}), source: "tdl" });
			}
			// The fence's first line follows the line its opening ``` is on; a
			// line past the fence's own, the appended block's, is the fence.
			const place = (line: number, column: number): Place => {
				const start = node.position?.start;
				if (!start || line < 1 || line > lines) return node.position;
				return { line: start.line + line, column: start.column - 1 + column };
			};

			const pkg = packageOf(node.value);
			if (pkg === undefined) fail("a tdl mermaid fence needs a package clause, which its target mermaid block names");

			const dir = await mkdtemp(join(tmpdir(), "remark-tdl-mermaid-"));
			try {
				await writeFile(join(dir, FENCE), withTarget(node.value, pkg));
				const result = await run(tdl, ["gen", "--target", "mermaid", "-o", "out", FENCE], dir);
				if ("failed" in result) fail(result.failed);
				const diags = diagnostics(result.stderr);

				for (const d of diags.filter((d) => d.warning)) {
					const message = report(d.message, place(d.line, d.column));
					if (d.code) message.ruleId = d.code;
				}
				if (!result.ok) {
					const [first, ...rest] = diags.filter((d) => !d.warning);
					for (const d of rest) report(d.message, place(d.line, d.column)).fatal = true;
					if (first) fail(first.message, place(first.line, first.column));
					fail(result.stderr.trim() || `${tdl} gen failed`);
				}

				const written = await readdir(join(dir, "out")).catch(() => []);
				const diagrams = written.filter((f) => f.endsWith(".mmd"));
				if (diagrams.length !== 1) {
					fail(`tdl gen wrote ${diagrams.length} Mermaid diagrams for one fence, where one was expected`);
				}
				const value = await readFile(join(dir, "out", diagrams[0] as string), "utf8");
				const drawn: Code = { type: "code", lang: "mermaid", meta: null, value: value.replace(/\n$/, "") };
				if (node.position) drawn.position = node.position;
				parent.children[index] = drawn;
			} finally {
				await rm(dir, { recursive: true, force: true });
			}
		}
	};
}
