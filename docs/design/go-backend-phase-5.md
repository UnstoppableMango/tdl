# Go backend phase 5: working plan

The implementation plan for phase 5 of [go-backend-plan.md](go-backend-plan.md), written to be picked up on another machine.
It is a working document: fold what it settles into [go-backend.md](go-backend.md) and [go-backend-plan.md](go-backend-plan.md), then delete it, in the pull request that lands phase 5.

## State

The Go backend work is a stack of branches, each with its own pull request:

| Branch | Pull request | Base | Holds |
| --- | --- | --- | --- |
| `fix/go-skip-cascade` | #781 | `main` | A declaration naming a skipped one is skipped too (closes #780) |
| `go-phase-3` | #782 | `main` | Generics and classes; carries #781's commit until it merges |
| `go-phase-4` | #787 | `go-phase-3` | `where` constraints as `Validate` methods |
| `go-phase-5` | none yet | `go-phase-4` | This plan |
| `go-phase-6` | none yet | `go-phase-5` | Units, not started |

`go-phase-7` (conformance) is added on top of `go-phase-6` with `gh stack add go-phase-7` when phase 6 lands.

Beside the stack:

- #786 (`fix/sema-include-keeps-constraints`, off `main`) keeps a field's constraints and default through `include` (closes #783).
- #784 (a `NAME` constraint argument is never resolved) and #785 (a generated file name can end in `_test` or a build constraint suffix) are open issues with no pull request.

## Before phase 5

1. **#781 has one open review thread, and it is valid.**
   `TestReferringToASkippedDeclarationSkipsItToo` reaches the skipped declaration only through an `Option`.
   Add fixtures reaching it through a collection element and through an alias, push a new commit to `fix/go-skip-cascade`, then reply to the thread and resolve it.
2. **Hercules fails on #782, #786, and #787, and not because of their changes.**
   `main` at `59f6b9f` fails Hercules evaluation too, while #781 on the same base passed.
   Its logs are not public, so this needs someone with Hercules access.
3. **`fix/sema-include-constraints` is a stray remote branch.**
   It holds the #786 commit with a broken message; #786 itself uses `fix/sema-include-keeps-constraints`.
   Delete it once its owner agrees.

## What phase 5 is

`foreign("github.com/acme/money", "Money")` in a target block maps a TDL declaration to an existing Go type, and the backend emits an import and a reference instead of a declaration.
It replaces the `decimal`, `uuid`, and `date` placeholders, and it is how an extern, a declaration in another package, is generated.
Phase 5 is done when a model naming a foreign type generates a package that imports it and does not redeclare it.

What exists:

- A target path can name a prelude primitive: `decimal => foreign(...)` resolves through the prelude scope and attaches to the prelude declaration (`internal/sema/target.go`, `attach`).
- The compiler checks directives on prelude declarations (`internal/gen/check.go`, `directivesFor`), and the Go backend never reads them.
- A target path cannot name an extern.
  `money.Money` and a name merged by a `_` import both fail with "names nothing", and `ir.Extern` has no directives field.
- [ir.md](ir.md) says an extern is mapped with a `foreign` directive, so the compiler has to change for that half.

## Pull request A, off `main`: a target path can name an extern

This is a compiler change any backend can use, so it is its own pull request.
Cherry-pick its commit onto `go-phase-5` so the backend work can use it before it merges.

- `proto/tdl/ir/v1/ir.proto`: `Extern` gains `repeated Directive directives = 4;`, which is additive, so `buf breaking` passes.
  Run `make generate` and commit `ir/ir.pb.go`.
- `internal/sema/target.go`, `attach`: a declaration binding still wins.
  Otherwise a head bound by a `_` import names that extern, and a head that is an import alias names `alias.Member` as an extern, interned with `l.extern`.
  A field path under an extern is an error, since the model cannot see an extern's fields.
  The same conflict ladder applies to an extern as to a declaration.
- `ir/dump.go` prints an extern's directives; regenerate the goldens with `go test ./internal/sema -update` and read the diff.
- `internal/gen/check.go`, `directivesFor`, includes extern directives, so the compiler checks them against the backend's handshake.
- `docs/spec.md`, target blocks: a path may name a declaration of an imported package, and such a path has no member.
  `docs/design/ir.md` says where the directive lands.
- Tests first in `internal/sema/lower_test.go`, and a conformance case with an import and a target entry for an extern.

## Pull request B, `go-phase-5`: `foreign` in the Go backend

### Decisions to confirm

1. **`foreign(path, name)`** is declared in the handshake with two string arguments.
   A declaration, a prelude primitive, or an extern carrying it for this target is written `alias.Name`, with type arguments when applied.
   An own declaration carrying it is not generated.
2. **The import is always aliased**, as in `import decimal "github.com/shopspring/decimal"`.
   The alias is the path's last element with a `.vN` suffix and a `go-` prefix dropped, made an identifier, and numbered when two paths in a file share one.
   The real package name cannot be known without loading the package, and an explicit alias makes the reference correct either way.
   A type parameter refuses a foreign alias's spelling as it refuses `fmt`.
3. **A foreign type is assumed comparable.**
   Go decides at the consumer's build, and refusing it would break `Set<uuid>` as soon as `uuid` is mapped.
   This is the one place the backend trusts the consumer, and the docs say so.
4. **A foreign type has no shape to check.**
   A standard constraint on one warns, validation does not visit it, and an own newtype marked foreign warns that its constraints go with the declaration.
   A mapped `uuid` stops being string-like, so `matches` on it warns rather than emitting a `string(x)` conversion that would not compile.
5. **An own declaration marked foreign carries nothing.**
   A key, a class marker, or `Validate` on one warns, since the backend declares nothing to put them on.

### Steps

Each test is written first and seen to fail.

1. Test helper: `files()` type checks through an importer that answers a foreign path with a stub package declaring the named types, since the source importer cannot load a module that is not on disk.
2. The handshake declares `foreign`, and `decimal => foreign(...)` makes a `decimal` field a `decimal.Decimal` with the import.
3. An own struct marked foreign is not generated and is referenced through its alias, including a generic application such as `Box[T]`.
4. An extern marked foreign is referenced; this needs pull request A.
   An extern with no mapping warns as it does today, and the message points at `foreign`.
5. Alias rules: a `.v3` suffix, a `go-` prefix, two paths with one last element, and a type parameter spelled like an alias.
6. Comparability, constraints, markers, keys, and validation on foreign types, following decisions 3 to 5.
7. `internal/gen/golang_test.go`: a foreign-mapped primitive in `goModel()`, so the hosts are held to agreeing on it.

Docs: in `go-backend.md`, the type mapping table and the placeholders paragraph, a Foreign types section, and Diagnostics; in `go-backend-plan.md`, phase 5 and the "What is not here" paragraph; the backend entry in `AGENTS.md`.

## Verification

- `go test -race ./...` and `go test -short ./...`, reading any golden diff after `-update`.
- End to end: a model importing a dependency file, with `decimal => foreign("github.com/shopspring/decimal", "Decimal")` and `money.Money => foreign("example.com/money", "Money")`.
  Run `tdl gen`, then `go build` and `go vet` in a scratch module whose `go.mod` replaces both paths with local stub modules.
- `buf lint`, `buf breaking` against `main`, `golangci-lint`, `markdownlint-cli2`, `nix fmt` with no changes, and `make test-treesitter`, since the grammar is unchanged.
- Pull request A against `main`; pull request B from `go-phase-5` against `go-phase-4`, saying it carries A's commit until A merges.
  Reply to every bot thread and resolve it.
