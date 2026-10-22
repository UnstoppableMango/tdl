# Implementing the TypeScript content mapper

An implementation plan for [typescript-content-mapper.md](typescript-content-mapper.md).
Phases are ordered by dependency, and each states what makes it done.

Phases 1 and 2 are done.

## Scope

This plan moves the host's directive checks and loss silencing below `internal/gen`, serves the `typescript` backend as a TypeScript 7.1 content mapper from `tdl-gen-typescript`, and publishes the npm package naming it.
`tdl gen --target typescript` writes the same bytes throughout.

Runtime values, platform packages carrying the binary, and unsaved dependency buffers are out of scope.

## Phase 1: `internal/host`

- `CheckDirectives` moves from `internal/gen/check.go` to `internal/host`, and `Silence` with it.
- `internal/host` reads `tdl.toml` through `internal/config` and returns the codes a target allows, as `internal/cli/gen.go` computes them today.
- `internal/gen` and `internal/cli` call it there; nothing else changes.

Done when `go test ./...` passes with no golden or recorded exchange changed, and `go list -deps ./internal/host` names no backend.

## Phase 2: the mapper

- Check the protocol against the 7.1 compiler: the shapes of `mappings`, `dependencies`, and a diagnostic's `category`, and correct the design where they differ.
  Done: mappings are arrays, there is no dependency list, and every diagnostic is an error.
- `backend/typescript/mapper.go`: JSON-RPC 2.0 with `Content-Length` framing, `initialize`, `openProject`, `transform`, and `closeProject`, lowering in process and running the backend in mapper mode.
- `extern.go` names an extern's module by the import's path relative to the importing file, in mapper mode only.
- Diagnostics as the design lists them, under the source `tdl`, with a code per kind; option diagnostics for any `options` key.
- `cmd/tdl-gen-typescript` serves the mapper under `content-mapper`, and the plugin with no argument.
- The pipe test and the `tsc` test in the design; the `tsc` test skips without a `tsc` of 7.1 or later.

Done when the pipe test passes over the smoke model and a two-file project, every diagnostic kind is asserted at its range, `TestHostsAgree` still passes, and the `tsc` test passes locally with 7.1.

## Phase 3: navigation

- The backend records the TDL range of each declaration, field, and variant it writes in mapper mode.
- `transform` returns them as `mappings`.

Done when a test asserts each mapping's range for the smoke model, and in VS Code with the TypeScript 7 extension, go to definition on an imported type lands on its TDL declaration.

## Phase 4: the npm package

- `npm/tdl-typescript/package.json` and its README; `release-please-config.json` versions it.
- `release-please.yml` gains a job publishing it with npm trusted publishing and `id-token: write`, gated on the release job's output, added to `unmango/actions` if missing.
- The trusted publisher for `@unmango/tdl-typescript` names `release-please.yml`; the scope's owner creates it before the first release.

Done when `npm pack` of the directory installs into the `tsc` test's project in place of the test's own package and the test passes, and a release publishes it without a stored token.

## Phase 5: docs and checks

- A section of `docs/backends/typescript.md` on importing `.tdl` files: setup, `import type`, the diagnostic codes, and the PATH requirement.
- `checks.typescript-content-mapper` once nixpkgs carries TypeScript 7.1; until then the phase stays open on that item.

Done when the page's setup works when copied into an empty project, and the check runs in `nix flake check`.
