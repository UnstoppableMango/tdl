// Finds a binary in the platform package npm installed beside @unmango/tdl,
// as esbuild finds its own. Exported for @unmango/tdl-typescript.

const path = require("node:path");

// process.platform and process.arch to the package carrying their binaries.
// nix/npm.nix writes one package per entry, and @unmango/tdl's
// optionalDependencies name each.
const packages = {
	"darwin arm64": "@unmango/tdl-darwin-arm64",
	"darwin x64": "@unmango/tdl-darwin-x64",
	"linux arm64": "@unmango/tdl-linux-arm64",
	"linux x64": "@unmango/tdl-linux-x64",
};

function binaryPath(name) {
	const key = `${process.platform} ${process.arch}`;
	const pkg = packages[key];
	if (!pkg) {
		throw new Error(
			`@unmango/tdl has no binaries for ${key}; it ships ${Object.keys(packages).join(", ")}. ` +
				"Install with Nix, or build with go install github.com/unstoppablemango/tdl/cmd/tdl@latest.",
		);
	}
	let manifest;
	try {
		manifest = require.resolve(`${pkg}/package.json`);
	} catch {
		throw new Error(
			`${pkg} is not installed. It is an optional dependency of @unmango/tdl, so an install that ` +
				"skipped optional dependencies (--omit=optional, or a lock file written on another platform) leaves it out.",
		);
	}
	return path.join(path.dirname(manifest), "bin", name);
}

module.exports = { binaryPath, packages };
