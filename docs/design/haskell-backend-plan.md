# Implementing the Haskell backend

An implementation plan for [haskell-backend.md](haskell-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the `haskell` backend.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

## Layout

```text
backend/haskell/       # the backend, package haskell, name "haskell"
cmd/tdl-gen-haskell/   # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
No Go library type checks Haskell, so the tests also run `ghc -fno-code -Wall -Werror` over every response when `ghc` is on `PATH`, as the TypeScript tests run `tsc`.
`checks.gen-haskell` always does, with `ghc` from nixpkgs, which carries every package the output imports.

`TestValidationRuns` builds the response with a `Main` module and runs it under `runghc`, since a check is shown to reject a value only by running it.
It skips under `-short` and when `runghc` is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: records, enums, and the type mapping

`Struct` in all three kinds, `Enum` as a sum type, `Newtype`, `Alias` as a synonym, and the primitive and collection type mapping.
The module header, its pragmas, explicit imports and the qualification rule, `module`, `name`, `prefix`, `lazy`, keyword escaping, constructor collisions, and the stock deriving set.
Haddock comments and `DEPRECATED` pragmas.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-haskell`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target haskell for smoke { out("haskell") }`, `nix/default.nix` gains `checks.gen-haskell`, and AGENTS.md gains a bullet under Backends.

Done when the smoke model generates one module that compiles with `-Wall -Werror`, and both hosts return byte-identical files.

## Phase 2: identity and foreign types

`key` as a function, and a key record for several fields.
`foreign` on a declaration, a primitive, or an extern, and an extern resolved through its dependency's `haskell` block.
`derive`.

Done when a two-package model generated into one `out` compiles as one set of modules, and a model mapping `uuid` to `Data.UUID` imports it and declares nothing for it.

## Phase 3: generics

Type variables, kind signatures, standalone deriving for a higher-kinded parameter, and the whole-model inference of `Functor`, `Foldable`, and `Traversable`.

Done when `Page<T>`, `Collection<f, T>`, and a generic enum compile, and `Page` derives `Functor` while a declaration holding its parameter in a set does not.

## Phase 4: classes

A class as a type class with `HasField` superclasses, an instance per satisfying declaration, multi-parameter classes, functional dependencies, associated types and their bindings, conditional instances, and class contexts.
`requires` on a type is written on the type's generated functions and warns.

Done when every class in the conformance corpus generates and compiles, an associated type binding included.
This is the phase where the Haskell backend generates what the Go backend warns about, so its tests carry a case for each row of [go-backend.md](go-backend.md#classes)'s warning table that Haskell expresses.

## Phase 5: validation

`validate<Name>`, `validate<Name>At`, and `mk<Name>` for each standard constraint but `matches`, which warns.
A container calls what it holds, at any depth, and a newtype checks its accumulated set.

Done when the standard constraint names generate a check that rejects a value violating them, under `TestValidationRuns`.

## Phase 6: units

Haskell can carry a unit in a type, which Go cannot, so the decision here is how much of the algebra to bring.

- **Phantom tags.** Each base and named unit is an empty data type, `data Kg`, and `decimal<kg>` is a newtype over the base type with a phantom parameter.
  A `decimal<kg>` cannot be passed as a `decimal<N>`, but a unit written as an expression has no name to tag with, as `Unit.decl` being unset says, and multiplying two quantities is the consumer's problem.
- **Type-level dimensions.** A unit is its reduced dimensions, as `ir.Unit` holds them, encoded as a type-level list, and a type family normalizes them.
  Every unit has a type, including an expression, but the output then needs a library, such as `dimensional`, or a module of type families the backend ships, which no other backend does.
- **Dropped.** A unit-typed field is its base type with a warning, as the Go backend's phase 6 also weighs.

Whichever is chosen, the declaration holding a unit-typed field is emitted, replacing phase 1's skip.

Done when a unit-typed field either generates a type that keeps its unit or warns that the unit was dropped, and the declaration holding it is emitted in both cases.

## Decisions deferred

- **A cabal file.** A `cabal` directive could write a library stanza listing the modules and their dependencies; until then the consumer's project lists them.
- **Regular expressions.** A `regex` directive naming a library, such as `regex-tdfa`, would let `matches` generate, with the pattern checked against that library's dialect at generation time.
- **Field prefixes.** A target-block directive writing `orderId` instead of `id`, for a consumer on a GHC older than 9.4 or one avoiding `OverloadedRecordDot`.
- **Lenses.** `Generic` already lets `generic-lens` reach every field, so nothing is generated for them.
