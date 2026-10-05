# Implementing the Go backend

An implementation plan for [go-backend.md](go-backend.md).
Phases are ordered by dependency, and each states what makes it done.

Phases 1 through 5 are done.
Phases 6 and 7 are not.

## Scope

This plan builds the `go` backend.
It speaks the protocol in [plugins.md](plugins.md) unchanged: if a phase needs something the protocol cannot express, the protocol changes, and the backend gets no private surface.

## Layout

```text
backend/golang/        # the backend, package golang, name "go"
cmd/tdl-gen-go/        # the same value served as a subprocess
```

A Go package cannot usefully be called `go`, so the directory is `golang`; `Name` is what a target block writes.

## Testing

The unit tests build `*ir.Model` values by hand with `backend/internal/irtest`, so a failure is the generator's and not the parser's or lowering's.

Generated Go has to compile.
Parsing is not enough, since `map[[]byte]struct{}` parses, so the unit tests type check the whole response as one package with `go/types` and a source importer.

`TestValidationRuns` builds the response in its own module and runs a test against it, since a check is shown to reject a value only by running it.
It skips under `-short` and when `go` is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: structs, enums, and the type mapping (done)

`Struct` in all three kinds, `Enum` in both shapes, `Newtype`, and the primitive and collection type mapping.
`Describe` declares `package`, `name`, and `tag`, and the output is one file per declaration, formatted with `go/format`.
Anything else is a positioned warning, and the declaration reaching it is skipped; a `where` constraint warns and the declaration is still emitted.
The backend is registered in `internal/gen/registry.go` and served by `cmd/tdl-gen-go`.

Done when a model containing an entity, a value, a mixin, both enum shapes, and a newtype generates Go that compiles, and both hosts return byte-identical files.

## Phase 2: identity (done)

An entity's `key` directive becomes a `Key()` method, returning the field itself when the directive names one and a generated `<Name>Key` struct when it names several.
[go-backend.md](go-backend.md#structs) says why, and a key that cannot be generated is a warning that leaves the entity without one.

Done when an entity's key is expressible in Go without the consumer reading the `.tdl` file.

## Phase 3: generics (done)

A `Param` becomes a Go type parameter, and a `ParamRef` becomes a use of it.
`comparable` is inferred from what reaches a map key.
A class is a marker interface with one unexported method each satisfying declaration carries, and a `requires` clause is a constraint naming it.
An interface of getters was rejected: the satisfying struct already declares the fields, and Go refuses a field and a method with one name.
[go-backend.md](go-backend.md#classes) has the rest, including what warns.

Done when a parameterized declaration generates a Go generic type that compiles, and a `requires` clause and its class either generate a constraint or warn saying why not.

## Phase 4: validation (done)

`where` constraints become a `Validate() error` method joining every violation, beside an unexported `validate` that threads a container's path.
An unknown constraint name warns and the rest is still checked.
A newtype's accumulated constraints run in its own method.
[go-backend.md](go-backend.md#validation) has what each standard name means.

Done when the standard constraint names generate a check that fails on a value violating them.

## Phase 5: foreign types (done)

`foreign("github.com/acme/money", "Money")` maps a TDL declaration, a primitive, or an extern to an existing Go type, and the backend emits an import and a reference rather than a declaration.
Every import is aliased, a foreign type carries no method, and it is assumed comparable.
[go-backend.md](go-backend.md#foreign-types) has the rest.

Done when a model naming a foreign type generates a package that imports it and does not redeclare it.

## Phase 6: units

A unit reaches the backend reduced to base dimensions, and Go has nothing that carries one.

The decision is whether a quantity becomes a named Go type, which keeps a `decimal<kg>` from being assigned to a `decimal<N>`, or whether the unit is dropped with a warning at the field.
A named type needs a name, and a unit written as an expression has none: `Unit.decl` is unset for exactly that case.
A named quantity's underlying type is whatever `foreign` maps `decimal` to.

Either way the declaration is emitted, replacing phase 1's skip, and the [Diagnostics](go-backend.md#diagnostics) section changes with it.

Done when a unit-typed field either generates a type that keeps its unit or warns that the unit was dropped, and the declaration holding it is emitted in both cases.

## Phase 7: conformance

A corpus for generated output, shaped like the parser and lowering corpora: a `.tdl` file, an expected tree of Go files, and a check that runs `go build` over the result.
It is plain text, so a Go backend written in another language could be held to it.

Done when a corpus case is a directory and adding one requires no code.

## Decisions deferred

- **Serialization.** No `encoding/json` opinion is generated; `tag` is how a consumer states one.
- **Comparability.** A set element, a map key, and a key field must each be comparable, and one that is not warns.
  A generated set type with methods would lift the restriction for sets only, and is a bigger commitment than a collection mapping should make on its own.
  A type parameter is not restricted: reaching a key makes it `comparable`, and the use supplying an incomparable argument warns.
- **A class as a field type.** It warns.
  An interface field holding any satisfying declaration is the obvious Go shape, and whether the language means that is the spec's call.
- **Validating a type argument's values**, and `Validate` on a sealed interface itself, which Go would need a function beside the interface for.
- **Doc comment rendering.** `///` comments become Go doc comments verbatim, without prefixing the identifier name, because rewriting a user's prose is worse than a vet warning.
- **One file per declaration.** One file per package is the alternative to revisit if a model with many small declarations makes the tree unreadable.
