# Implementing anonymous unions

An implementation plan for [unions.md](unions.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the union syntax, its lowering to a synthesized enum, the wire rule, and the native mapping in every shipped backend, then reading unions back where a target has them.
Backends that are designed and not built take the mapping from [unions.md](unions.md#targets) when their enum phase lands; this plan does not amend their plans.

Type parameters as alternatives, a per-union discriminant, and structural or literal alternatives are deferred in the design and out of scope.

## Testing

Conformance cases under `testdata/conformance/union-*` cover the syntax and lowering: a two-alternative union, flattening through an alias, a shared union in two fields, `(A | B)?`, `A | B | null`, and a union in a list.
Each starts with a `pending` file that the phase earning it deletes.
`testdata/invalid/` takes the parse errors (`A | B?`, `A? | B`, `null | A`); lowering diagnostics on files that parse are Go tests in `internal/sema`, as the [ir plan](ir-plan.md#testing) puts them.

`testdata/gen/smoke/source.tdl` gains a union of each wire shape, so every `checks.gen-*` runs its language's tool over a union.
Each backend phase adds a round-trip case under `testdata/roundtrip/<target>/union/`.

## Phase 1: syntax

`docs/grammar.ebnf` and `docs/spec.md` gain `Alternative` and `ParenType`; `ast.TypeRef` gains `Union []*TypeRef` and `Paren *TypeRef`, with `ast.Children` and `TestChildrenCoverEveryField` following.
`parser.parseTypeRef` reads alternatives, and `parseTypeArg` tells a parenthesized type from a unit group by the first operator inside it.
`ast.Fprint` prints `A | B` with single spaces, and `TestCorpusIsCanonical` covers the new cases.
`make treesitter` and `make textmate` regenerate; a type reference after `|` and `(` is colored as one after `:`.

Done when every union case parses, prints canonically, and parses in tree-sitter with no ERROR node, and the invalid cases fail with their golden.

## Phase 2: lowering and ir

`proto/` gains `SYNTACTIC_FORM_UNION` and `Enum.union`; `make generate`.
`internal/sema` flattens, expands aliases, deduplicates, and interns a union by its sorted alternative set, synthesizing one `Enum` declaration per set with the joined name and sorted variants.
It checks the [rules](unions.md#rules): arity, no parameter, no class or unit, `?` on a lone alternative, `null` last, colliding variant or declaration names, and the field `name` agreement.
The recursion check treats the synthesized enum as any enum.
`ir.Dump` prints it, and `internal/unlower` prints a union reference as written and drops its declaration.

The wire shape table lives in a new `internal/wire`, read by `sema` for the distinguishability check and, from phase 3, by `emit`.

Done when the union conformance cases lower to their goldens, `TestCorpusRoundTrips` passes over them, and each rule has a Go test with one positioned diagnostic.

## Phase 3: emit and the language server

`backend/internal/emit` gains `Union`, which reports whether an enum is a union and, per variant, its alternative's type and whether it is tagged, read from `internal/wire`.
`Resolve` returns a union as a new `Form`, so a backend switches on it as it does on `Option`.
`irtest` gains a union builder.

Every shipped backend's existing fielded-enum path is checked against a union with a test, so before phase 4 a union already generates something that compiles in every target.

The language server hovers on `|` and completes types after it.

Done when every backend's tests pass with a union in the model and `checks.gen-*` pass on the smoke file.

## Phase 4: JSON-native targets

TypeScript writes a native union, JSON Schema and OpenAPI 3.x a `oneOf`, each with the tagged alternatives intersected with their discriminant.
OpenAPI writes a `discriminator` when every alternative is tagged, and 2.0 warns.
The JSON Schema instance tests gain union instances: a tagged object, a bare scalar, and a rejected ambiguous value.

Done when `tsc --strict`, the JSON Schema instance tests, and `vacuum lint` pass on the smoke file, and the TypeScript output reads as the design's example.

## Phase 5: nominal targets

Go writes a sealed interface that declared alternatives implement and the rest are wrapped for, with `MarshalJSON` and `UnmarshalJSON` under `json` following the wire rules.
Haskell writes a sum type with prefixed constructors.

Done when `go vet` and `ghc -Wall -Werror` pass on the smoke file and a Go test round-trips each wire shape through `encoding/json`.

## Phase 6: schema IDLs

protobuf writes a `oneof`, Thrift and Smithy a `union`, and GraphQL a union with bare alternatives wrapped.
Salesforce warns on a field holding a union; LikeC4 writes nothing new.

Done when protocompile, thriftgo, `smithy validate`, and gqlparser accept the smoke file's output.

## Phase 7: import

TypeScript, Smithy, Thrift, and GraphQL read their native unions back as unions where [unions.md](unions.md#import) says they can; Go and Haskell read them from `roundtrip` annotations.
`TestConformanceComesBack` covers the union conformance cases through each importing target.

Done when every `testdata/roundtrip/<target>/union/` case passes.

## Decisions deferred

- Whether a union's synthesized name should also be overridable in a target block, once target paths can name something anonymous.
- Whether `tdl fmt` sorts alternatives. The model ignores order, so sorting is safe; leaving the author's order is the default.
