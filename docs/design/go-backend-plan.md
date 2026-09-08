# Implementing the Go backend

An implementation plan for [go-backend.md](go-backend.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds the `go` backend: the first code generator in this repository.

[plugins.md](plugins.md) covers the protocol and not what a backend does with it: a code generator is a body of work with its own plan and its own conformance story.
This is that plan.
The protocol it speaks is finished and unchanged by anything here; if a phase needs something the protocol cannot express, the protocol is what changes, and this plan does not get a private surface.

`ir` is finished enough to generate from: lowering covers the whole grammar and the conformance corpus lowers with no diagnostic.

## Layout

```text
backend/golang/        # the backend, package golang, name "go"
cmd/tdl-gen-go/        # the same value served as a subprocess
```

The directory is `golang` and the backend is called `go`.
`Name` is what a target block writes; a Go package cannot usefully be called `go`.

## What is not here

**`foreign`.** Three primitives map to a placeholder because the right answer is a dependency the backend must not choose.
Phase 5 is where that is fixed, and until then a project wanting `decimal.Decimal` post-processes.

**Dependency target blocks.** `ir-plan.md` phase 8b is not done, so a model does not see a dependency's blocks.
Nothing here is blocked on it, because a generated package refers to a dependency through an extern either way.

## Testing

The unit tests build `*ir.Model` literals by hand, the way `backend/debug/debug_test.go` does.
A hand-built model states exactly the shape under test and does not drag the parser and lowering into a generator's failure.

The end-to-end test is the only one that matters: generated Go has to compile.
A test that asserts on substrings can pass while emitting something `go build` refuses, and every phase here adds a shape that could.
Parsing is not enough either, since `map[[]byte]struct{}` parses, so the unit tests type check the whole response as one package with `go/types` and a source importer.

Both hosts run the same assertions, in process and as a subprocess, following `internal/gen/subprocess_test.go`.
That is the protocol's invariant and this backend is the second thing holding it.

## Phase 1: structs, enums, and the type mapping

The shapes that need no new machinery: `Struct` in all three kinds, `Enum` in both of the shapes [go-backend.md](go-backend.md) splits it into, `Newtype`, and the primitive and collection type mapping.

`Describe` declares `package`, `name`, and `tag`.
The output is one file per declaration, formatted with `go/format`.

Everything else, a type parameter, a unit, a class, an extern, or a set element or map key Go cannot compare, is a warning with a position and the declaration reaching it is skipped.
A `where` constraint warns and the declaration is still emitted, since skipping a constrained newtype would leave every field naming it undeclared.

The backend is registered in `internal/gen/registry.go` and served by `cmd/tdl-gen-go`.

Done when a model containing an entity, a value, a mixin, both enum shapes, and a newtype generates Go that compiles, and when the in-process and subprocess hosts return byte-identical files.

## Phase 2: identity

`key` fields become something.
An entity's identity is the one thing the language says an entity has that a value does not, and phase 1 emits them as ordinary fields, which loses it.

The candidates are a `Key()` method returning a comparable struct, a generated key type per entity, and nothing at all with the `key` bits exposed as metadata.

Done when an entity's key is expressible in Go without the consumer reading the `.tdl` file.

## Phase 3: generics

Type parameters reach the backend unmonomorphized, which `ir` did on purpose so a language with generics emits them.

A `Param` becomes a Go type parameter, and a `ParamRef` in a field type becomes a use of it.
The open question is constraints: a `requires` clause names a TDL class, and a class is not a Go interface.

Done when a parameterized declaration generates a Go generic type that compiles, and when a `requires` clause either generates a constraint or produces a warning saying why it cannot.

## Phase 4: validation

`where` constraints become code.

The decisions are where the check lives (a `Validate() error` method is the obvious one), what an unknown constraint name does, and whether a newtype's accumulated constraints run in one place or per inherited origin.
The constraint name set is open, so the backend understands what it understands and warns about the rest.

Done when the standard constraint names generate a check that fails on a value violating them.

## Phase 5: foreign types

`foreign("github.com/acme/money", "Money")` in a target block maps a TDL declaration to an existing Go type, and the backend emits an import and a reference rather than a declaration.

This is what makes the `decimal`, `uuid`, and `date` placeholders survivable, and it is also how an extern is generated: a declaration in another package is a foreign type whose mapping the consumer supplies.

Done when a model naming a foreign type generates a package that imports it and does not redeclare it.

## Phase 6: conformance

A corpus for generated output, in the shape the parser and lowering corpora already have: a `.tdl` file, an expected tree of Go files, and a check that runs `go build` over the result.

The corpus is plain text and not Go code, like the others, so a Go backend written in another language could be held to it.

Done when a corpus case is a directory and adding one requires no code.

## Decisions deferred

- **Serialization.** No `encoding/json` opinion is generated.
  `tag` is how a consumer states one, and a backend that emitted tags by default would be choosing a wire format on their behalf.
- **`Set` as a map.** `map[T]struct{}` is what phase 1 emits.
  A generated set type with methods is nicer to use and is a bigger commitment than a collection mapping should make on its own.
  It would also lift the comparability restriction, which is the strongest argument for it.
- **Doc comment rendering.** `///` comments become Go doc comments verbatim.
  Whether a name is prefixed to satisfy Go's "comment starts with the identifier" convention is left alone, because rewriting a user's prose is worse than a vet warning.
- **One file per declaration.** Settled in [go-backend.md](go-backend.md), but the alternative of one file per package is the thing to revisit if a model with many small declarations makes the tree unreadable.
