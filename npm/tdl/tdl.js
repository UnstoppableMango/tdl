#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const { binaryPath } = require("./lib/resolve.js");

let bin;
try {
	bin = binaryPath("tdl");
} catch (err) {
	console.error(err.message);
	process.exit(1);
}

// tdl finds a plugin such as tdl-gen-mermaid on PATH, so the platform
// package's own come first.
const env = { ...process.env, PATH: [path.dirname(bin), process.env.PATH].filter(Boolean).join(path.delimiter) };
const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit", env });
if (result.error) {
	console.error(`${bin}: ${result.error.message}`);
	process.exit(1);
}
if (result.signal) {
	process.kill(process.pid, result.signal);
}
process.exit(result.status ?? 1);
