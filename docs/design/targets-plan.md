# Target blocks at scale: plan

An implementation plan for [targets.md](targets.md).
The two bugs come first, then the phases in order of how much each removes from a real model's target blocks, and each states what makes it done.

No phase is done.

## Scope

The plan changes the grammar, lowering, the IR, the protobuf, thrift, jsonschema, openapi, and typescript backends, and the language server.
unmango/apis is the model each phase is measured against: a phase is worth its cost when it removes lines from apis's target blocks without changing a byte of the generated protos.

## Testing

Syntax and lowering are conformance cases: a `source.tdl` and an `ir.golden` per construct, and an `error.golden` per invalid case.
`TestCorpusIsCanonical` holds every new file to `tdl fmt`, and `make treesitter` and `make textmate` regenerate the derived grammars after each grammar change.
Backend phases add golden cases to the backend's tests and a block to `testdata/gen/smoke/source.tdl`, so the nix checks run each language's tool on the output.

## Phase 1: references are not containment

The protobuf backend stops writing a non-`owned` entity field as the entity's message.
`reference(T)` names the message that carries a reference; without it the field is a `string` with a warning.
`resource(type, pattern)` on an entity writes `(google.api.resource)`, and a reference to it gains `(google.api.resource_reference).type`.

This changes output for any model that relied on the old behavior, so it lands as a `fix!` with a release note.

Done when a reference in `testdata/gen/smoke` generates the configured reference message with its `resource_reference`, an `owned` entity field still embeds the message, and apis's `Commit.repository` can be written `repository: Repository` with identical output.

## Phase 2: several directives on one path

`TargetEntry`'s `=>` form takes one or more directives, and a block-scope directive after an entry is an error (#922).
The parser needs two tokens of lookahead to end a directive list; `docs/grammar.ebnf` states the rule in a comment, since EBNF cannot.
`ast.Fprint` keeps a path's directives on one line when they fit.

Done when the #922 reproduction applies both directives to the field, a late block-scope directive fails with an error naming the entry, and the corpus is unchanged.

## Phase 3: access in the model

Undecided: how a field's access is spelled, and how that relates to `owned`, are open ([targets.md](targets.md#access)).
This phase is written once that is settled.
It adds the syntax, carries access on `ir.Field`, and teaches `internal/unlower` to print it.

## Phase 4: presence and access conventions

Each backend in the [conventions table](targets.md#conventions) writes presence, and access once phase 3 lands.
jsonschema, openapi, and typescript do so by default; protobuf under `field_behavior("google")` or `field_behavior("marked")`.
A collection without `length(1..)` counts as absent-able.

Done when `testdata/gen/smoke` generates `OPTIONAL` in protobuf and leaves a `T?` field out of `required` in JSON Schema and OpenAPI, an output-only field generates `readOnly`, `readonly`, and `OUTPUT_ONLY` once phase 3 lands, and an explicit `field_behavior` option wins with a warning.

## Phase 5: field selectors

A path segment may be `*`, and a class or kind selector may be followed by a field segment.
Specificity follows [targets.md](targets.md#field-selectors); a selector matching nothing is an error.
If [profiles-plan.md](profiles-plan.md) has landed, its phase 1 restriction on field segments is relaxed to the cases targets.md allows; if not, this phase adds kind selectors to target blocks alone and profiles inherit them.

Done when `*.page_token` and `Entity.conditions` reach every matching field in a conformance case, a named field beats a selector in the golden, and an unmatched selector fails.

## Phase 6: wire identity

A leading integer numbers a field or variant, and `reserved` followed by a number or string lists retired numbers and names.
Lowering reports a duplicate number and a reserved number as errors, which `tdl check` and the language server show (#991).
`ir.Field.number`, `ir.Variant.number`, and `ir.Struct.reserved` carry them; `emit.Numbers` reads the model's number before the `number` directive, and refuses both on one member.
protobuf and thrift write a `reserved` member's doc comment above the statement.

Done when apis's `Commit` numbered in the model generates the same `commit.proto`, a duplicate number fails `tdl check`, and `TestCorpusRoundTrips` round-trips numbers.

## Phase 7: editors

The language server hovers a node's resolved directives per target, shows allocated numbers as inlay hints, resolves each target path segment and selector for go to definition, and offers a code action numbering a field (#992).
It builds on the per-segment positions [lsp-editors-plan.md](lsp-editors-plan.md) phase 8 plans; if that has not landed, this phase adds them.

Done when hovering `Commit.summary` in apis shows its number and options, and go to definition from `Commit.summary` in a target block lands on the field.

## Migrating unmango/apis

Each phase lands in apis as its own pull request, regenerating with no diff in `proto/unmango`:

1. Typed references and `resource`, package by package.
1. Late `edition(...)` lines move to the top of each target block.
1. Fields gain `?` and modifiers, and protobuf switches to `field_behavior("marked")`; once the protobuf importer (#963) lands, it can propose each field's access from the existing options.
1. Field numbers move into the model, and the target blocks shrink to `foreign`, `resource`, `file`, and the few options with no convention.
