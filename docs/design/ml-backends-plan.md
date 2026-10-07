# Implementing the ML backends

An implementation plan for [ml-backends.md](ml-backends.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the shared core and the `ocaml`, `reason`, `sml`, and `fsharp` backends.
They speak the protocol in [plugins.md](plugins.md) unchanged.

OCaml comes first because it exercises the most of the core: the ordering, the module plan for classes, and the validation plan.
Reason follows because it is a printer over the same tree, and so proves the split between core and dialect early.
Standard ML reuses the module plan, and F# replaces it with interfaces and adds units, so it comes last.

## Layout

```text
backend/internal/ml/   # the core: the plan, the syntax tree, and Dialect
backend/ocaml/         # package ocaml, name "ocaml"
backend/reason/        # package reason, name "reason"
backend/sml/           # package sml, name "sml"
backend/fsharp/        # package fsharp, name "fsharp"
cmd/tdl-gen-ocaml/     # each backend served as a subprocess
cmd/tdl-gen-reason/
cmd/tdl-gen-sml/
cmd/tdl-gen-fsharp/
```

## Testing

The core's tests assert on the plan, the groups and what each declaration becomes, not on text, so one test covers four dialects.
Each backend's tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text, then run the compiler named in [Checking the output](ml-backends.md#checking-the-output) over every response when it is on `PATH`.
Each target has a `checks.gen-<name>` that always does, over `testdata/gen/smoke`, with the compiler from nixpkgs.

`TestValidationRuns` in each backend runs the response with a main that feeds it invalid values, since a check is shown to reject a value only by running it.
It skips under `-short` and when the runner is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds each backend's in-process and subprocess hosts to byte-identical output.

## Phase 1: the core and OCaml types

`backend/internal/ml` with the syntax tree, the `Dialect`, the strongly connected components and their order, and the expansion of an alias inside a group.
`backend/ocaml` with records, enums, newtypes, and aliases, the type mapping, sets and maps as functor applications with their `compare_` and `equal_` functions and recursive modules for a self-holding group, names and escaping, the predefined-type rule, the module name, doc comments, and deprecation.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-ocaml`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target ocaml for smoke { out("ocaml") }`, `nix/default.nix` gains `checks.gen-ocaml`, and AGENTS.md gains a bullet under Backends.

Done when the smoke model generates one module that compiles with warnings as errors, a model with two mutually recursive entities generates one `and` group, an entity holding a set of itself compiles as recursive modules, a set of records holding sets keeps one copy of equal elements, and both hosts return byte-identical files.

## Phase 2: OCaml identity, foreign types, and derive

`key` as a function and a key record, `foreign` on a declaration, a primitive, or an extern, an extern resolved through its dependency's `ocaml` block, and `derive`.

Done when a two-package model generated into one `out` compiles as two modules, and `instant => foreign("Ptime", "t")` imports nothing else and declares nothing for `instant`.

## Phase 3: OCaml generics and classes

Type variables, the arrow-kind warning, and `requires` on a type.
Classes as module types, instances as modules, conditional instances as functors, multi-parameter classes, associated types and their bindings, and class contexts as includes.

Done when every class in the conformance corpus generates or warns, and the spec's conditional instance compiles as a functor applied in a test to an auditable type.

## Phase 4: OCaml validation

`validate_<name>`, `validate_<name>_at`, and `make_<name>` for each standard constraint but `matches`, which warns.
A container calls what it holds, at any depth, a newtype checks its accumulated set, and the functions follow the core's recursive groups.

Done when each standard constraint is rejected under `TestValidationRuns`.

## Phase 5: Reason

`backend/reason`: the Reason printer, its names and escapes, and extern resolution reading an `ocaml` block as well as a `reason` one.
Every OCaml test model is generated in Reason too.

Done when the smoke model and the class cases convert with `refmt --print ml` and compile, and a Reason model naming an OCaml dependency's type compiles against it.

## Phase 6: Standard ML

`backend/sml`: its type mapping, records as one-constructor datatypes, the constructor-collision warning, top-level signatures and functors around the model's structure, the `result` datatype, the UTF-8 length helper, and sets and maps through the SML/NJ library's functors with generated `compare` functions.

Done when the smoke model and the class cases pass `mlton -stop tc` with the SML/NJ library on the `.mlb`, and validation runs under MLton.

## Phase 7: F\#

`backend/fsharp`: its type mapping, `namespace rec`, companion modules for validation, keys as members, `attribute`, classes as interfaces with the warnings ml-backends.md lists, `requires` on a type as a constraint, units of measure, `matches` through `Regex`, `#nowarn "44"`, and an extern resolved through a `csharp` block.
The .NET pieces come from `backend/internal/dotnet`, shared with [csharp-backend-plan.md](csharp-backend-plan.md); whichever plan reaches them first creates the package.

Done when the smoke model, the class cases, and a model with derived units load in `dotnet fsi` with warnings as errors, `decimal<N>` and `decimal<kg*m/s^2>` are one F# type in a test that assigns one to the other, and an F# model naming a C# dependency's type builds against it.

## Phase 7b: F# serialization

`json("stj")` and `discriminant` through FSharp.SystemTextJson, with the wire convention read from `backend/internal/dotnet`.
This follows phase 5 of csharp-backend-plan.md, which states the convention.

Done when a test serializes a value of every smoke declaration and validates it against the `jsonschema` backend's schema for the smoke model, and a value C# serialized deserializes in F# to an equal one.

## Phase 8: shipped profiles

`prelude/std/ocaml.tdl`, `reason.tdl`, and `fsharp.tdl`, with `yojson`, `ptime`, and `stj` from [Where profiles fit](ml-backends.md#where-profiles-fit).
This waits on phase 3 of [profiles-plan.md](profiles-plan.md).

Done when each target's smoke check also runs once with each of its profiles applied, and the output still compiles with the libraries the profile names.

## Decisions deferred

- **Build files.** A `dune`, `.fsproj`, or `.mlb` directive listing the module and its dependencies; until then the consumer's build lists them.
- **Interface files.** OCaml's `.mli` would let a newtype's constructor be private; every export is public today, as in Haskell.
- **Lenses and accessors.** Records are read with the language's own field syntax, and no accessor is generated outside a class's module.
