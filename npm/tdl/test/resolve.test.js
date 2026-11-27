// The resolver names a platform package and a binary for every platform
// nix/npm.nix packs. node --test runs it in checks.npm-tdl.

const assert = require("node:assert/strict");
const { test } = require("node:test");
const { executable, packageFor, packages } = require("../lib/resolve.js");

test("each platform names its package by npm's os and cpu", () => {
	for (const [platform, arch, pkg] of [
		["darwin", "arm64", "@unmango/tdl-darwin-arm64"],
		["darwin", "x64", "@unmango/tdl-darwin-x64"],
		["linux", "arm64", "@unmango/tdl-linux-arm64"],
		["linux", "x64", "@unmango/tdl-linux-x64"],
		["win32", "arm64", "@unmango/tdl-win32-arm64"],
		["win32", "x64", "@unmango/tdl-win32-x64"],
	]) {
		assert.equal(packageFor(platform, arch), pkg);
	}
	assert.equal(Object.keys(packages).length, 6);
});

test("a platform with no package names the ones shipped", () => {
	assert.throws(() => packageFor("freebsd", "x64"), /no binaries for freebsd x64; it ships .*win32 x64/);
});

test("windows binaries end in .exe", () => {
	assert.equal(executable("tdl", "win32"), "tdl.exe");
	assert.equal(executable("tdl-gen-typescript", "win32"), "tdl-gen-typescript.exe");
	assert.equal(executable("tdl", "linux"), "tdl");
	assert.equal(executable("tdl", "darwin"), "tdl");
});
