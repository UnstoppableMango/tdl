# Implementing the Rust backend

An implementation plan for [rust-backend.md](rust-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the `rust` backend.
The class plan, the wire convention, `Length`, and `Pattern` come from `backend/internal/emit`, which every backend shares.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

## Layout

```text
backend/rust/          # the backend, package rust, name "rust"
cmd/tdl-gen-rust/      # the same value served as a subprocess
nix/rust/Cargo.lock    # the crates the output may name, for the checks
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
The whole-model decisions, which fields are boxed and which types derive `Ord`, are tested as decisions rather than as text.
No Go library type checks Rust, so the tests also run `cargo clippy --offline -- -D warnings` over every response, in a throwaway workspace using `nix/rust/Cargo.lock`, when `cargo` is on `PATH` and the locked crates are in the local registry.
`checks.gen-rust` always does, over `testdata/gen/smoke`, with the crates vendored by `rustPlatform.importCargoLock`, and also runs `cargo fmt --check`.

`TestValidationRuns` builds the response with a test file that feeds each constructor invalid values, since a check is shown to reject a value only by running it.
It skips under `-short` and when `cargo` cannot build offline.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: the crate, structs, enums, and the type mapping

`Cargo.toml` and `src/lib.rs`, the crate name from the package or `crate`, the header and its lint allowances, and the manifest listing only the crates the output names.
Structs in all three kinds with public fields, fieldless and fielded enums, unconstrained newtypes, a newtype over a newtype holding its parent, aliases as `pub type`, and the primitive, collection, and absence mappings.
The boxing rule over the whole model and `boxed`; the derive decision over the whole model, and a set element or map key holding a float skipped.
Absolute paths, `name`, snake-case fields, raw identifiers, the collisions Rust refuses, doc comments, and `#[deprecated]`.
A `where` constraint warns that it is not enforced yet, and a unit-typed field is its base type with a warning.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-rust`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target rust for smoke { out("rust") }`, `nix/default.nix` gains `checks.gen-rust`, and AGENTS.md gains a bullet under Backends.

Done when the smoke model generates a crate that passes Clippy with warnings as errors, a self-referencing value through an optional builds with exactly one `Box`, and both hosts return byte-identical files.

## Phase 2: identity, defaults, foreign types, and the manifest

`key` as a method and a key struct, defaults as associated functions and `impl Default`, `foreign` on a declaration, a primitive, or an extern, an extern resolved through its dependency's `rust` block with a `path` dependency, `dependency`, `derive`, `attribute`, and `layout("module")`.

Done when a two-package model generated into two crates builds as one workspace, `date => foreign("chrono", "NaiveDate")` with `dependency("chrono", "0.4")` builds against chrono, and the smoke model under `layout("module")` builds as a module of a crate the test writes.

## Phase 3: generics and classes

Type parameters and `requires` as bounds, restated on every generated impl, and the higher-kind warning.
`tdl::Entity` and its impls for own and extern entities.
Classes through `emit.PlanInterfaces`: traits with getters and supertraits, an impl per satisfying declaration, separate and conditional instances as impls, multi-parameter classes as trait parameters, functional dependencies as associated types, associated types and their bindings, and the warning for a second dependency.

Done when every class in the conformance corpus generates and builds, an associated type binding and a conditional instance included, `Envelope<i64>` fails to build in a test where `Envelope<T> requires Auditable<T>`, and a second `Projection` instance for one type fails to build where the class has `from -> to`.
This is the phase where the Rust backend generates what the Go and C# backends warn about, so its tests carry a case for each row of [go-backend.md](go-backend.md#classes)'s warning table that Rust expresses.

## Phase 4: constraints

Checked newtypes with a private field, `new`, `check`, `get`, `into_inner`, and the conversions; a type per constrained field, named for its struct and field or by `newtype`; `tdl::Violations` and its messages.
Each standard constraint, `length` counted by `chars()` from `emit.Length`, NaN failing `min` and `max`, the oneOf enum and the excluded-argument warning, and `length(N)` on bytes as an array.
Defaults on checked fields checked at generation time.
`matches` checked by `emit.Pattern` and printed in Rust syntax, each compiled once in a `LazyLock`.
If the Java or .NET pattern printer exists by then, the walk moves into `backend/internal/emit` with a table per engine and Rust fills in its table; otherwise this phase creates it there.

Done when each standard constraint is rejected under `TestValidationRuns`, no generated type has a public way to hold a violating value, a string of four emoji passes `length(4)`, `\d` refuses `٣`, `\b` treats `é` as a non-word character, `[a&&b]` matches `&`, and a pattern RE2 refuses warns at generation time.

## Phase 5: serialization

`serde`: the derives and every attribute in the design's table, the `tdl::base64` and `tdl::double_option` adapters, checked newtypes decoding through `TryFrom`, and the `serde` features on the crates the output names.
The discriminant and tags come from `emit.Discriminant` and `emit.Tag`, and `discriminant` renames the property.

Done when a test serializes a value of every smoke declaration with `serde_json` and validates the output against the `jsonschema` backend's schema for the smoke model, deserializes it back to an equal value, and refuses a payload that omits a `T | null` field or holds a value violating a constraint.

## Phase 6: units

`Measure` over the model's base units, an alias per named unit, `Add` and `Sub` between measures of one dimension, and a `unit` parameter as one const parameter per base dimension, replacing phase 1's warning.
This phase decides what a unit-typed field crossing two packages with different base units becomes: a warning and the base type, or a shared order of base units read from the import graph.

Done when `decimal<N>` and `decimal<kg*m/s^2>` generate one type, a test assigning a `decimal<kg>` to a `decimal<m>` field fails to build, and the smoke model's units build.

## Phase 7: shipped profiles

`prelude/std/rust.tdl` with `chrono` and `time`, each mapping `instant`, `date`, and `duration` and adding the dependency, and `serde`, setting the directive.
This waits on phase 3 of [profiles-plan.md](profiles-plan.md).

Done when the smoke check also runs once with each profile applied, and the output builds with the crates the profile names.

## Decisions deferred

- **A runtime crate.** Publishing `tdl` as a crate, so `Entity` and `Measure` cross generated crates.
- **Ranged integers.** A const-generic range type for `min` and `max` on integers, in place of a field type per constrained field.
- **References.** `Arc` or keys for non-owned entity fields, behind a directive.
- **Reverse.** A plugin in Rust over `syn`, reading this backend's output back to TDL.
