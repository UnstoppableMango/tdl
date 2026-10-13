# Implementing the Zod backend

An implementation plan for [zod-backend.md](zod-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the `zod` backend.
The wire convention, `Length`, `Pattern`, and the class plan come from `backend/internal/emit`, which every backend shares.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

## Layout

```text
backend/internal/ts/   # what typescript and zod share: primitives, names, reserved words, extern paths
backend/zod/           # the backend, package zod, name "zod"
cmd/tdl-gen-zod/       # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
No Go library type checks TypeScript, so the tests also run `tsc --noEmit --strict --exactOptionalPropertyTypes` over every response, and run it under `node`, when `tsc`, `node`, and the `zod` package are found, as the `typescript` tests run `tsc`.
They skip under `-short` and otherwise.
`checks.gen-zod` always runs them, with `zod` from a fixed-output npm fetch.

Some tests are a `tsc` run that must fail: a snippet beside the response that the compiler must reject, since the design's claim is that the type refuses it.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: the module, structs, enums, and the type mapping

`backend/internal/ts` takes the primitive mapping, names, reserved words, property quoting, and the extern import path out of `backend/typescript`, which reads them from there with no change to its output.
The `.zod.ts` file, the header, `const` and `type` pairs in source order, and getters with return types for forward and recursive references.
`z.object` for all three struct kinds, `closed` as `z.strictObject`, `z.enum` for a fieldless enum, `z.discriminatedUnion` of variant objects for a fielded one with `discriminant`, aliases expanded, and newtypes as unbranded schemas of their base.
The primitive, collection, and absence mappings, with `instant` requiring seconds, with `unique` declared once, and the integer map key settled one way or the other.
`name`, JSDoc and `@deprecated`, and the collisions.
A `where` constraint warns that it is not enforced yet, and a unit-typed field is its base type with a warning.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-zod`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target zod for smoke { out("zod") }`, `nix/default.nix` gains `checks.gen-zod`, `backend/internal/roundtrip`'s `targets` table gains a row with no normal form, and AGENTS.md gains a bullet under Backends.

Done when the smoke model generates a module that `tsc` accepts in strict mode, a mutually recursive pair of entities type checks and parses, and the agreement test passes: each inferred type and the `typescript` backend's type for the same declaration are assignable to each other.

## Phase 2: defaults, externs, and foreign types

`.default()` checked at generation time, `foreign` on a declaration or an extern, and an extern resolved through its dependency's `zod` block by the path the shared package computes.

Done when a two-package model generated into one `out` type checks and parses as two modules, and a `foreign` schema is imported and not declared.

## Phase 3: constraints

The literal unions and tuples, with `narrow.go`'s rule moved into `backend/internal/emit` and read by both `typescript` and `zod`; `typescript` writes a small exact list length as a tuple in the same change.
Brands on every newtype and on a field's schema named for its struct and field or by `newtype`, stacked for a newtype over a newtype, and `structural` writing none.
Each standard constraint by the design's table, with `codePoints` and `size` declared once, the messages naming the constraint, and the excluded-argument warning.

`matches` checked by `emit.Pattern` and printed by a walk over `regexp/syntax` in `backend/internal/emit`, with the ECMAScript table: the `u` flag, folds as classes, every class as ranges, the line anchors as lookaround, and named captures.
All of it only under the `patterns` directive: without it a `matches` warns and is not checked, and with it the exponential-backtracking shape still warns and drops the check.
If the Rust, Java, .NET, or Python printer exists by then, Zod adds a table to that walk; otherwise this phase creates it.

Done when each standard constraint refuses a value under `node`, `LineItem` with `quantity: 0` and a plain `string` where a `Sku` is expected fail `tsc`, the agreement test still passes under `structural`, a string of four emoji passes `length(4)`, a `matches` without `patterns` warns and emits no `.regex()`, `(a+)+$` and `(a|aa)+$` warn under `patterns`, and the pattern table test agrees with Go's `regexp` on every input, including `\s` refusing a no-break space, `\p{Greek}` accepting `λ`, `(?i)k` accepting the Kelvin sign, and `^.$` accepting one emoji.

## Phase 4: codecs

`codec` on the target block: `instant`, `bytes`, sets, and maps as the design's table decodes them, brands applied to the decoded value, and the agreement test reading `z.input`.

Done when a value of every smoke declaration decodes from JSON the `jsonschema` backend's schema accepts, encodes back to equal JSON, and a duplicate in a set and an instant with microseconds are refused on decode.

## Phase 5: generics and classes

Generics for `typescript` and `zod` together, since both write the TypeScript type: a schema function per parameterized declaration and a written type beside it, `requires` as a bound on the schema parameter's output, and the higher-kind warning.
Classes through `emit.PlanInterfaces` as structural types with the warnings the design lists.

Done when every class and generic declaration in the conformance corpus generates and checks, `Page(Order)`'s inferred type equals `Page<Order>`, and `Envelope(z.int())` fails `tsc` where `Envelope<T> requires Auditable<T>`.

## Phase 6: units

A unit-typed number or decimal string branded with its base dimensions, as `ir` interns them.

Done when `float64<N>` and `float64<kg*m/s^2>` are assignable to each other, `float64<m>` is not assignable to either, and the declarations holding them are emitted.

## Decisions deferred

- **JSON Schema patterns.** Printing `jsonschema` and `openapi` patterns through the ECMAScript table, a change to those backends once phase 3 builds it.
- **Linear-time patterns.** An `engine("linear")` directive adding V8's `l` flag, once V8's linear engine accepts the `u` flag the printed patterns carry.
- **Zod Mini.** A `flavor("mini")` directive printing the functional spelling.
- **Readonly output.** `.readonly()` on every object, at the cost of agreement with `typescript`.
- **Reverse.** Through `z.toJSONSchema` and a JSON Schema importer, ordered by [reverse-plan.md](reverse-plan.md).
