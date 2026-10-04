// Runs inside the editor. Each check asks VS Code the question a person
// would, through the same commands the editor's own UI uses, so a pass means
// the extension started the server and the editor reached it.

import * as assert from "node:assert/strict";
import * as vscode from "vscode";

const source = `package p

primitive string

/// An address to write to.
type Email: string

type User: Entity { id: string email: Email nope: Nope }
`;

export async function run(): Promise<void> {
	const folder = vscode.workspace.workspaceFolders?.[0];
	assert.ok(folder, "the editor opened no workspace");

	const uri = vscode.Uri.joinPath(folder.uri, "user.tdl");
	await vscode.workspace.fs.writeFile(uri, new TextEncoder().encode(source));
	const doc = await vscode.workspace.openTextDocument(uri);
	await vscode.window.showTextDocument(doc);
	assert.equal(doc.languageId, "tdl");

	const diagnostics = await waitFor(() => {
		const found = vscode.languages.getDiagnostics(uri);
		return found.length > 0 ? found : undefined;
	});
	assert.ok(
		diagnostics.some((d) => d.message.includes("Nope")),
		`no diagnostic names Nope: ${diagnostics.map((d) => d.message).join("; ")}`,
	);

	const hovers = await vscode.commands.executeCommand<vscode.Hover[]>(
		"vscode.executeHoverProvider",
		uri,
		doc.positionAt(at("email: Email") + "email: ".length),
	);
	const text = hovers
		.flatMap((h) => h.contents)
		.map((c) => (typeof c === "string" ? c : c.value))
		.join("\n");
	assert.ok(text.includes("type Email: string"), `hover does not show the declaration:\n${text}`);
	assert.ok(text.includes("An address to write to."), `hover does not show the doc comment:\n${text}`);

	// VS Code narrows the server's whole-document edit into minimal ones
	// before handing them back, so the edits are applied and the result is
	// what is compared.
	const edits = await vscode.commands.executeCommand<vscode.TextEdit[]>("vscode.executeFormatDocumentProvider", uri, {
		tabSize: 2,
		insertSpaces: true,
	});
	assert.ok(edits && edits.length > 0, "formatting a file with a one-line body produced no edit");
	const apply = new vscode.WorkspaceEdit();
	apply.set(uri, edits);
	assert.ok(await vscode.workspace.applyEdit(apply), "the formatting edits did not apply");
	assert.ok(
		doc.getText().includes("type User: Entity {\n  id: string\n  email: Email\n  nope: Nope\n}\n"),
		`formatted text is not canonical:\n${doc.getText()}`,
	);
}

// at is where needle starts in the source, failing rather than returning -1,
// which positionAt would quietly read as the start of the file.
function at(needle: string): number {
	const i = source.indexOf(needle);
	assert.ok(i >= 0, `${needle} is not in the source`);
	return i;
}

// waitFor polls until get returns a value, since diagnostics arrive when the
// server publishes them rather than in answer to anything.
async function waitFor<T>(get: () => T | undefined, timeoutMs = 30_000): Promise<T> {
	const deadline = Date.now() + timeoutMs;
	for (;;) {
		const value = get();
		if (value !== undefined) {
			return value;
		}
		if (Date.now() > deadline) {
			throw new Error(`nothing arrived within ${timeoutMs}ms`);
		}
		await new Promise((resolve) => setTimeout(resolve, 100));
	}
}
