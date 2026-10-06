# Implementing the LikeC4 backend

An implementation plan for [likec4-backend.md](likec4-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.
Phases 1 through 4 need nothing from [profiles.md](profiles.md); phase 5 needs [profiles-plan.md](profiles-plan.md) phases 1 through 3.

## Scope

This plan builds the `likec4` backend.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

## Layout

```text
backend/likec4/        # the backend, package likec4, name "likec4"
cmd/tdl-gen-likec4/    # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
No Go library parses LikeC4, so validation is `checks.gen-likec4`, which runs the `likec4` CLI.
`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: elements

`Backend`, `Describe`, `tdl.c4`, the package element, and one element per declaration with its kind, title, and description.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-likec4`, and is listed in `nix/cmd.nix`.

Done when `TestHostsAgree` passes and the smoke model gives one element per declaration.

## Phase 2: relationships

The type walk, field relationships, `composes`, `includes`, variant fields, and a newtype's base.

Done when every field of the smoke model naming a declaration gives exactly one relationship, and a mixin gives one `includes` however many fields it contributes.

## Phase 3: classes, services, and lifting

`conforms`, the `service`, `rpc`, and `stream` tags, `element`, `edge`, and lifting through omitted declarations.

Done when the model in `backend/protobuf/service_test.go`, with requests and responses omitted in a target block, draws `accepts` and `returns` relationships from the service to the entities its messages hold.

## Phase 4: names, externs, and views

Id rules and the keyword list, `name`, `title`, `technology`, `link`, and `id`, relationships into a dependency, the package view, and `views(false)`.
`testdata/gen/smoke/source.tdl` gains `target likec4 for smoke { out("likec4") }`.
`flake.nix` gains the `github:unmango/pkgs` input, and `nix/default.nix` gains `checks.gen-likec4`.
AGENTS.md gains a bullet under Backends.

Done when `nix flake check` validates the smoke output, and a two-package model generated into one `out` validates as one project.
The check confirms that LikeC4 draws an `index` view on its own; if it does not, `tdl.c4` declares `view index { include * }`.

## Phase 5: shipped profiles

`std.likec4.entities` and `std.likec4.services` in `prelude/std/likec4.tdl`.
`checks.gen-likec4` generates and validates the smoke model once per profile.

Done when both profiles validate and each drawing has the elements its profile keeps and no others.
