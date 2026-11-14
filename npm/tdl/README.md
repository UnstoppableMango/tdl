# @unmango/tdl

The [Type Description Language](https://github.com/UnstoppableMango/tdl) compiler.
TDL describes a domain model once, and `tdl gen` writes it as TypeScript, Zod, Go, protobuf, JSON Schema, and the other targets the project lists.

```shell
npm install --save-dev @unmango/tdl
npx tdl gen shop.tdl
```

The package carries no binary itself.
npm installs the one platform package that matches the machine, from the `optionalDependencies`, and `tdl` runs its binary:

- `@unmango/tdl-linux-x64`: Linux, x86-64
- `@unmango/tdl-linux-arm64`: Linux, ARM64
- `@unmango/tdl-darwin-x64`: macOS, Intel
- `@unmango/tdl-darwin-arm64`: macOS, Apple silicon

No package runs an install script, so `--ignore-scripts` changes nothing.
An install that leaves out optional dependencies, with `--omit=optional` or a lock file written on another platform, leaves `tdl` without a binary, and it says so.

Each platform package also carries `tdl-gen-typescript`, which the TypeScript content mapper runs, and `tdl-gen-mermaid`, the `mermaid` backend, which `tdl` finds beside itself when run through the package.
`tdl-gen-mermaid` is a Bun single executable, so on Linux it needs glibc.

On another platform, build from source with `go install github.com/unstoppablemango/tdl/cmd/tdl@latest`, or use the Nix flake.
