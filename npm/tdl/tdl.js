#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const { binaryPath } = require("./lib/resolve.js");

let bin;
try {
	bin = binaryPath("tdl");
} catch (err) {
	console.error(err.message);
	process.exit(1);
}

const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
	console.error(`${bin}: ${result.error.message}`);
	process.exit(1);
}
if (result.signal) {
	process.kill(process.pid, result.signal);
}
process.exit(result.status ?? 1);
