// Starts `tdl lsp` for .tdl files and restarts it when tdl.server.path
// changes.

import { constants } from "node:fs";
import { access } from "node:fs/promises";
import * as path from "node:path";
import { type ExtensionContext, window, workspace } from "vscode";
import { LanguageClient, type LanguageClientOptions, type ServerOptions } from "vscode-languageclient/node";

let client: LanguageClient | undefined;

// Every start and stop runs through one chain, so overlapping configuration
// changes restart the server in order. A failed step does not block the
// next.
let lifecycle: Promise<void> = Promise.resolve();

function enqueue(step: () => Promise<void>): Promise<void> {
	lifecycle = lifecycle.then(step, step);
	return lifecycle;
}

export async function activate(context: ExtensionContext): Promise<void> {
	context.subscriptions.push(
		workspace.onDidChangeConfiguration((event) => {
			if (event.affectsConfiguration("tdl.server.path")) {
				enqueue(async () => {
					await stop();
					await start();
				}).catch(report);
			}
		}),
	);
	await enqueue(start);
}

export async function deactivate(): Promise<void> {
	await enqueue(stop);
}

function report(err: unknown): void {
	const reason = err instanceof Error ? err.message : String(err);
	void window.showErrorMessage(`TDL: the language server did not stop cleanly: ${reason}`);
}

async function start(): Promise<void> {
	const configured = workspace.getConfiguration("tdl").get<string>("server.path") || "tdl";

	// Look the executable up here: the client reports a missing one as a
	// connection failure and retries.
	const command = await resolve(configured);
	if (command === undefined) {
		void window.showErrorMessage(
			`TDL: no language server at "${configured}". Install tdl or set tdl.server.path; highlighting still works.`,
		);
		return;
	}

	const server: ServerOptions = { command, args: ["lsp"] };
	const options: LanguageClientOptions = {
		documentSelector: [{ scheme: "file", language: "tdl" }],
	};

	const next = new LanguageClient("tdl", "TDL", server, options);
	try {
		await next.start();
		client = next;
	} catch (err) {
		const reason = err instanceof Error ? err.message : String(err);
		void window.showErrorMessage(`TDL: the language server at "${command}" did not start: ${reason}`);
	}
}

async function stop(): Promise<void> {
	const running = client;
	client = undefined;
	await running?.stop();
}

// resolve finds an executable the way a shell would: a name with a
// separator is used as written, and a bare name is searched for on PATH.
async function resolve(name: string): Promise<string | undefined> {
	const candidates = name.includes(path.sep) || name.includes("/") ? [name] : onPath(name);
	for (const candidate of candidates) {
		try {
			await access(candidate, constants.X_OK);
			return candidate;
		} catch {
			// Not here; try the next.
		}
	}
	return undefined;
}

function onPath(name: string): string[] {
	const dirs = (process.env.PATH ?? "").split(path.delimiter).filter((dir) => dir !== "");
	const names = process.platform === "win32" ? [name, `${name}.exe`] : [name];
	return dirs.flatMap((dir) => names.map((n) => path.join(dir, n)));
}
