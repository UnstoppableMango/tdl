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

Phase 4 is the first to run generated code: `TestValidationRuns` builds the response in a module of its own and runs a test against it, since a check is shown to reject a value only by running it.
It skips under `-short` and when `go` is not on `PATH`.

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

An entity's key becomes something.
An entity's identity is the one thing the language says an entity has that a value does not, and phase 1 emits an entity as an ordinary struct, which loses it.

Which fields identify an entity is a target directive, `LineItem => key(order, sku)` in a `go` block, rather than language syntax, so this phase reads the directive.
The key is a `Key()` method: it returns the field itself when the directive names one, and a generated `<Name>Key` struct when it names several.
[go-backend.md](go-backend.md#structs) says why, and a key that cannot be generated is a warning and leaves the entity without one.

Done when an entity's key is expressible in Go without the consumer reading the `.tdl` file.

## Phase 3: generics

Type parameters reach the backend unmonomorphized, which `ir` did on purpose so a language with generics emits them.

A `Param` becomes a Go type parameter, and a `ParamRef` in a field type becomes a use of it.
`comparable` is inferred from what reaches a map key, since TDL has no way to write it.

The question was constraints: a `requires` clause names a TDL class, and a class is not a Go interface until the backend makes it one.
A class declaration is the same question from the other side, since whatever a `requires` clause generates is what the class it names has to become.

A class is a marker interface, with one unexported method each satisfying declaration carries, and a `requires` clause is a constraint naming it.
An interface of getters was the alternative and loses: the satisfying struct declares the fields already, Go refuses a field and a method with one name, and generated code never calls into a type argument's values, so a getter would serve nothing beyond saying which types may be arguments, which the marker says.
[go-backend.md](go-backend.md#classes) has the rest, including what warns.

Left to later work: a conditional instance, a class taking parameters, an instance for a foreign type (phase 5), and a class as a field type.
A multi-parameter class like `Projection<from, to>` has no Go shape at all, because its content is a relationship between types.

Done when a parameterized declaration generates a Go generic type that compiles, and when a `requires` clause and the class it names either generate a constraint or produce a warning saying why they cannot.

## Phase 4: validation

`where` constraints become code.

A check is a `Validate() error` method joining every violation, beside an unexported `validate` that threads a container's path, so a nested violation names where it is.
An unknown constraint name warns and the rest is still checked, since the set is open.
A newtype's accumulated constraints run in one place, the newtype's own method, because the compiler already hands it the whole set.
[go-backend.md](go-backend.md#validation) has what each standard name means.

Left to later work: validating a type argument's values, and `Validate` on a sealed interface itself, which Go would need a function beside the interface for.

Done when the standard constraint names generate a check that fails on a value violating them.

## Phase 5: foreign types

`foreign("github.com/acme/money", "Money")` in a target block maps a TDL declaration to an existing Go type, and the backend emits an import and a reference rather than a declaration.

This is what makes the `decimal`, `uuid`, and `date` placeholders survivable, since a primitive is a declaration a target block names like any other.

Every import is aliased, because a package's name is not always its path's last segment and nothing in the model says which it is, so the qualifier is true by construction.
A foreign type carries no method: Go declares one beside the type, so a `where` constraint on a mapped declaration and a class it satisfies each warn.
[go-backend.md](go-backend.md#foreign-types) has the rest.

Left to later work: an extern, which is the same mapping for a declaration an imported TDL package owns.
`attach` in `internal/sema/target.go` resolves a target path against the model's own declarations, so there is nothing for a mapping to attach to yet, and that is a compiler change rather than a backend one.

Done when a model naming a foreign type generates a package that imports it and does not redeclare it.

## Phase 6: units

A unit reaches the backend reduced to base dimensions, and Go has nothing that carries one.

The decision is whether a quantity becomes a named Go type, which keeps a `decimal<kg>` from being assigned to a `decimal<N>`, or whether the unit is dropped with a warning at the field.
A named type needs a name, and a unit written as an expression rather than declared has none: `Unit.decl` is unset for exactly that case.
This follows phase 5 because a named quantity's underlying type is whatever `foreign` maps `decimal` to.

Either way the declaration is emitted, as it is for a `where` constraint the backend cannot check, since the unit is what is missing and not the type.
That replaces phase 1's skip, and the Diagnostics section of [go-backend.md](go-backend.md#diagnostics) changes with it.

Done when a unit-typed field either generates a type that keeps its unit or produces a warning saying the unit was dropped, and the declaration holding it is emitted in both cases.

## Phase 7: conformance

A corpus for generated output, in the shape the parser and lowering corpora already have: a `.tdl` file, an expected tree of Go files, and a check that runs `go build` over the result.

The corpus is plain text and not Go code, like the others, so a Go backend written in another language could be held to it.

Done when a corpus case is a directory and adding one requires no code.

## Decisions deferred

- **Serialization.** No `encoding/json` opinion is generated.
  `tag` is how a consumer states one, and a backend that emitted tags by default would be choosing a wire format on their behalf.
- **Comparability.** A set element, a map key, and a key field must each be a comparable Go type, and one that is not is a warning.
  `map[T]struct{}` is what phase 1 emits for `Set`, and a generated set type with methods would lift the restriction for sets, which is the strongest argument for it.
  It is nicer to use and a bigger commitment than a collection mapping should make on its own.
  `Map` keys and key fields stay restricted either way, since a generated set type does not change what `map[K]V` or `map[LineItemKey]LineItem` accepts.
  A type parameter is not restricted: reaching a key makes it `comparable`, and the use supplying an argument Go cannot compare is what warns.
- **A class as a field type.** Naming a class as a field's type warns.
  An interface field holding any satisfying declaration is the obvious Go shape, and whether the language means that is the spec's call rather than a backend's.
- **Doc comment rendering.** `///` comments become Go doc comments verbatim.
  Whether a name is prefixed to satisfy Go's "comment starts with the identifier" convention is left alone, because rewriting a user's prose is worse than a vet warning.
- **One file per declaration.** Settled in [go-backend.md](go-backend.md), but the alternative of one file per package is the thing to revisit if a model with many small declarations makes the tree unreadable.
