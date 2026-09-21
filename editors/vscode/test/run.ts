// Launches an editor with the extension under development and runs
// suite.ts inside it. The editor is VS_CODE, an executable already
// installed, rather than one @vscode/test-electron downloads: a downloaded
// build is linked against a loader NixOS does not have.
//
// TDL is the tdl the extension starts, written into the throwaway
// profile's settings so the run cannot pick up another from PATH.

import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { runTests } from "@vscode/test-electron";

async function main(): Promise<void> {
	const editor = required("VS_CODE");
	const tdl = required("TDL");

	const root = path.resolve(__dirname, "..", "..");
	const work = await mkdtemp(path.join(tmpdir(), "tdl-vscode-"));
	try {
		const user = path.join(work, "user");
		await mkdir(path.join(user, "User"), { recursive: true });
		await writeFile(path.join(user, "User", "settings.json"), JSON.stringify({ "tdl.server.path": tdl }));

		const workspace = path.join(work, "workspace");
		await mkdir(workspace);

		await runTests({
			vscodeExecutablePath: editor,
			extensionDevelopmentPath: root,
			extensionTestsPath: path.join(root, "dist", "test", "suite.js"),
			launchArgs: [workspace, "--user-data-dir", user, "--disable-extensions", "--disable-workspace-trust"],
		});
	} finally {
		await rm(work, { recursive: true, force: true });
	}
}

function required(name: string): string {
	const value = process.env[name];
	if (!value) {
		throw new Error(`${name} is not set`);
	}
	return value;
}

main().catch((err: unknown) => {
	console.error(err);
	process.exit(1);
});
