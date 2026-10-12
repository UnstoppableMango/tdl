# Implementing the Python backend

An implementation plan for [python-backend.md](python-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the `python` backend.
The class plan, the wire convention, `Length`, and `Pattern` come from `backend/internal/emit`, which every backend shares.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

## Layout

```text
backend/python/        # the backend, package python, name "python"
cmd/tdl-gen-python/    # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
No Go library type checks Python, so the tests also run `mypy --strict --enable-error-code=exhaustive-match` and Pyright in strict mode over every response, and import it under `python3`, when each is on `PATH`, as the TypeScript tests run `tsc`.
`checks.gen-python` always does, with `python313`, `mypy`, `pyright`, `ruff`, and `pydantic` from nixpkgs, and also runs `ruff format --check` and `ruff check`.

Some tests are a checker run that must fail: a snippet beside the response that the checker must reject, such as `LineItem(quantity=0)` or a non-exhaustive `match`, since the design's claim is that the checker refuses it.

`TestValidationRuns` imports the response with a script that feeds each constructor invalid values, since a check is shown to reject a value only by running it.
It skips under `-short` and when `python3` is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: the module, structs, enums, and the type mapping

The module path from the package or `module`, the `__init__.py` layout, the header and its aliased imports, and the order that puts a base class before its subclasses.
Frozen, slotted, keyword-only dataclasses in all three struct kinds, a field reaching a map left out of the hash, `StrEnum` for a fieldless enum, a union of variant classes behind a `type` alias for a fielded one, subclass and wrapper newtypes, a newtype over a newtype, and aliases as `type` statements.
The primitive, collection, and absence mappings, `Unset`, and a set element or map key reaching a map skipped.
`name`, `prefix`, snake-case fields, screaming-snake members, keyword escaping, the collisions, docstrings, and `warnings.deprecated`.
The fixed-width integers are plain `int` until phase 4, a `where` constraint warns that it is not enforced yet, and a unit-typed field is its base type with a warning.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-python`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target python for smoke { out("python") }`, `nix/default.nix` gains `checks.gen-python`, `backend/internal/roundtrip`'s `targets` table gains a row with no normal form, and AGENTS.md gains a bullet under Backends.

Done when the smoke model generates a module that passes both checkers in strict mode and `ruff check`, a `match` over a fielded enum missing a variant fails the checker, and both hosts return byte-identical files.

## Phase 2: identity, defaults, and foreign types

`key` as a method and a key class, defaults on fields with a factory for maps, `foreign` on a declaration, a primitive, or an extern, and an extern resolved through its dependency's `python` block.

Done when a two-package model generated into one `out` imports and checks as two modules, and `instant => foreign("pendulum", "DateTime")` imports `pendulum` and declares nothing for it.

## Phase 3: generics and classes

PEP 695 type parameters, `requires` as a bound, a declared protocol for a parameter with two bounds, and the higher-kind warning.
Classes through `emit.PlanInterfaces`: protocols with read-only properties and their superclasses, multi-parameter classes as generic protocols, and the warnings the design's table lists.

Done when every class in the conformance corpus generates and checks, `Envelope[int]` fails the checker where `Envelope<T> requires Auditable<T>`, `Envelope[Order]` passes both checkers where `Order` satisfies `Auditable` and fails them where the test removes one of the class's fields from `Order`.

## Phase 4: constraints

The constraints the checker states: `Literal` for `oneOf` and small integer ranges, and a fixed-length tuple for a small exact `length`, with the rule moved from `backend/typescript/narrow.go` into `backend/internal/emit` so both backends read it.
Checked types: subclass and wrapper newtypes with checking constructors and `check`, a type per constrained field named for its struct and field or by `newtype`, the `ValueError` with one note per violation, and the checked `Int32`, `Uint32`, `Int64`, and `Uint64` with `width("unchecked")`.
Each standard constraint, NaN failing `min` and `max`, the excluded-argument warning, and defaults on checked fields checked at generation time.
`matches` checked by `emit.Pattern` and printed in `re` syntax, with Unicode classes as ranges, case folding as classes, and `\Z` for end of text, each compiled once in a module constant.
If the Rust, Java, or .NET pattern printer exists by then, Python fills in a table for the shared walk; otherwise this phase creates the walk in `backend/internal/emit`.

Done when each standard constraint is rejected under `TestValidationRuns`, `LineItem(quantity=0)` and `Order(currency="GBP")` fail the checker, a string of four emoji passes `length(4)`, `\d` refuses `٣`, `^[a-z]+$` refuses `"abc\n"`, `\p{Greek}` accepts `λ`, and a pattern RE2 refuses warns at generation time.

## Phase 5: serialization

`json("pydantic")`: Pydantic dataclasses, aliases carrying the model's spelling with the config that keeps the Python name in the constructor, the discriminated union and each variant's `kind` field, `__get_pydantic_core_schema__` on every checked type, `AwareDatetime` for `instant`, base64 bytes, and the serializer leaving out an absent `T?` and an unset `T? | null`.
The discriminant and tags come from `emit.Discriminant` and `emit.Tag`, and `discriminant` renames the property.

Done when a test serializes a value of every smoke declaration and validates the output against the `jsonschema` backend's schema for the smoke model, decodes it back to an equal value, and refuses a payload that omits a `T | null` field or holds a value violating a constraint.

## Phase 6: units

Python's checker cannot reduce a unit, so the decision here is how much of one to keep.

- **A nominal type per unit and base.** `decimal<kg>` is `class KgDecimal(Decimal)`, named from the unit declaration that reduces to the field's dimensions, so a `decimal<kg>` cannot be passed where a `decimal<m>` is expected. Arithmetic returns the base type, and a unit expression no declaration reduces to has no name to give its type.
- **`Annotated` metadata.** `Annotated[Decimal, Unit(kg=1)]` keeps every unit, expressions included, for a consumer's library to read, and no checker enforces it.
- **Dropped.** Phase 1's warning stays.

Done when a unit-typed field either generates a type that keeps its unit or warns that the unit was dropped, and the declaration holding it is emitted in both cases.

## Decisions deferred

- **A pyproject file.** A directive could write `pyproject.toml` and `py.typed` for an `out` that is its own distribution; until then the consumer's project carries them.
- **Linear-time patterns.** `regex("re2")` and the `google-re2` binding.
- **A runtime module.** Publishing `tdl` as a Python package, so `Unset` and the checked integers cross generated modules.
- **Reverse.** An embedded `outline.py` over Python's `ast` module, read as `outline.js` is for TypeScript, ordered by [reverse-plan.md](reverse-plan.md).
- **Shipped profiles.** `prelude/std/python.tdl` with `pydantic`, setting `json("pydantic")`, once phase 3 of [profiles-plan.md](profiles-plan.md) is done.
