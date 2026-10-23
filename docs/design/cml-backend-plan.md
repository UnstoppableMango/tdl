# Implementing the CML backend

An implementation plan for [cml-backend.md](cml-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.
Phase 1 starts from the prototype [contextmapper-spike.md](contextmapper-spike.md) left in `backend/contextmapper`, which is unregistered.

## Scope

This plan builds the `cml` backend.
It speaks the protocol in [plugins.md](plugins.md) unchanged, and imports nothing.

## Layout

```text
backend/contextmapper/                   # the backend, package contextmapper, name "cml"
backend/contextmapper/testdata/cmlcheck/ # cml-check: Main.java and its script
cmd/tdl-gen-cml/                         # the same value served as a subprocess
nix/cml-check.nix                        # cml-check, built against context-mapper-cli's jars
```

## Testing

The unit tests lower TDL sources and assert on the generated text, and run `cml-check` on every response when it is on `PATH`, failing on any error or warning it prints.
`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.
`checks.gen-cml` validates the smoke model's output whether or not `cml-check` is on a developer's `PATH`.

## Phase 1: elements and aggregates

What the prototype does, registered: domain objects, fields, primitives, constraints, aggregate inference, keywords, and `name`.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-cml`, and is listed in `nix/cmd.nix`.
`backend/contextmapper/features.toml` holds the cells a measurement cannot, and `docs/backends/cml.md`'s table becomes measured.
`testdata/gen/smoke/source.tdl` gains `target cml for smoke { out("cml") }`.
`nix/cml-check.nix` compiles `cml-check` with `javac` against the `context-mapper-cli` package from `github:unmango/pkgs`; `nix/default.nix` gains `checks.gen-cml`, and CI's race run puts `cml-check` on `PATH`.
AGENTS.md gains a bullet under Backends.

Done when `TestHostsAgree` passes, `checks.gen-cml` validates the smoke output with no error or warning, and `TestTablesAreCurrent` measures the table.

## Phase 2: bounded context and aggregate directives

`context`, `type`, `vision`, `responsibilities`, and `technology` on the target block, the package doc as the vision statement, and `aggregate` on a declaration, with each `DirectiveSpec`.

Done when a model placing two unowned entities in one aggregate with `aggregate` validates with one `aggregateRoot`, and every bounded context property appears as CML spells it.

## Phase 3: context map

Imports and externs: `import` with the path both blocks' `out` directives place, the `ContextMap` holding this context and its direct dependencies, references into a dependency, and `upstream`, `supplier`, `partner`, and `kernel`.
A dependency with no `cml` block drops each field naming it with `lossy.unsupported`.

Done when a chain of three packages generated into one `out` validates file by file, each relationship directive writes its roles, and the Import and extern row generates.

## Phase 4: services

A type tagged `service` becomes a `Service` in the aggregate its operations touch, or on the bounded context when they touch several; each `rpc` field is an operation.

Done when the model in `backend/protobuf/service_test.go` validates as CML with one service and an operation per rpc.
