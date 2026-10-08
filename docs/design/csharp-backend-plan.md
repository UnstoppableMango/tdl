# Implementing the C# backend

An implementation plan for [csharp-backend.md](csharp-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds `backend/internal/dotnet` and the `csharp` backend.
The class plan, the wire convention, `Length`, and `Pattern` come from `backend/internal/emit`, which every backend shares.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

`backend/internal/dotnet` is shared with the `fsharp` target of [ml-backends-plan.md](ml-backends-plan.md).
Whichever plan reaches its .NET work first creates the package; the other imports it, and moving a rule into it moves that rule's tests with it.

## Layout

```text
backend/internal/dotnet/   # shared with fsharp: primitives, namespaces, foreign, attribute, the regex engine, json
backend/csharp/            # package csharp, name "csharp"
cmd/tdl-gen-csharp/        # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
`backend/internal/dotnet`'s tests assert on decisions, such as a primitive's base class library type, so one test covers both targets.
No Go library type checks C#, so the tests also run `dotnet build` over every response, in a throwaway project with `Nullable` enabled and warnings as errors, when `dotnet` is on `PATH`.
`checks.gen-csharp` always does, over `testdata/gen/smoke`, with `dotnet-sdk` from nixpkgs.

`TestValidationRuns` builds the response with a console project that feeds it invalid values, since a check is shown to reject a value only by running it.
It skips under `-short` and when `dotnet` is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: records, enums, and the type mapping

`backend/internal/dotnet` with the primitive mapping and the namespace derived from a package.
`backend/csharp` with records, fieldless and fielded enums, newtypes as record structs, aliases expanded, the collection mapping, optionality through nullability and `required`, generated equality and `TdlEquality`, the file header, `namespace`, `file`, `name`, names and collisions, doc comments, and `Obsolete`.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-csharp`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target csharp for smoke { out("csharp") }`, `nix/default.nix` gains `checks.gen-csharp`, and AGENTS.md gains a bullet under Backends.

Done when the smoke model builds with warnings as errors, two orders holding equal item lists compare equal in a test, and both hosts return byte-identical files.

## Phase 2: identity, defaults, foreign types, and attributes

`key` as a method and a key record struct, defaults as property initializers, `foreign` on a declaration, a primitive, or an extern, an extern resolved through its dependency's `csharp` block, and `attribute`.
`foreign` and `attribute` are parsed and validated in `backend/internal/dotnet`.

Done when a two-package model generated into one project builds, `instant => foreign("NodaTime", "Instant")` builds against NodaTime, and a defaulted field omitted from a constructing expression holds its default.

## Phase 3: generics and classes

Type parameters, `requires` as a `where` constraint, the higher-kind and unit-kind warnings, and the optional-bare-parameter warning.
Classes through `emit.PlanInterfaces`, with C#'s `InterfaceRules` for the multi-parameter rows; then C#'s interfaces, base lists, and `IProjection<TTo>` for a class with a dependency.

Done when every class in the conformance corpus generates or warns and builds, and `Envelope<int>` fails to build in a test where `Envelope<T> requires Auditable<T>`.

## Phase 4: validation

`Validate`, `ValidateAt`, `Create`, and `TryCreate` for each standard constraint, the content checks reaching through containers at any depth, and the accumulated set on a newtype.
`length` from `emit.Length`, counted by runes, and `matches` checked by `emit.Pattern` and printed in .NET syntax by `backend/internal/dotnet`.

Done when each standard constraint is rejected under `TestValidationRuns`, a string of four emoji passes `length(4)`, `\d` refuses a non-ASCII digit, `\p{Lu}` accepts `Ä`, and a pattern RE2 refuses or a script class warns at generation time.

## Phase 5: serialization

`json("stj")` and `discriminant`: property names, string enums, polymorphic fielded enums, and newtype converters.
The discriminant and tags come from `emit.Discriminant` and `emit.Tag`, which phase 7b of ml-backends-plan.md reads for `fsharp` too.

Done when a test serializes a value of every smoke declaration with `System.Text.Json` and validates the output against the `jsonschema` backend's schema for the smoke model, and deserializes it back to an equal value.

## Phase 6: shipped profiles

`prelude/std/csharp.tdl` with `stj`, setting `json("stj")`, and `nodatime`, mapping `instant`, `date`, and `duration`.
This waits on phase 3 of [profiles-plan.md](profiles-plan.md).

Done when the smoke check also runs once with each profile applied, and the output builds with the packages the profile names.

## Decisions deferred

- **Project files.** A `.csproj` directive, decided together with F#'s `.fsproj`.
- **Serializer contexts.** A `JsonSerializerContext` under `json("stj")`, for trimming and AOT.
