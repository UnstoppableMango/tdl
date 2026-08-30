# The ir package

Design document.
Nothing here is implemented; the pipeline currently stops at `ast`.

`ir` is the resolved semantic model: what `ast` becomes after names are resolved, sugar is lowered, and instances are checked.
It is the only thing a backend sees.
Two consumers share it, in-process backends importing the Go package, and plugin executables reading a protobuf off stdin.

## What resolution does

`ast` mirrors source one-to-one and leaves every name unresolved.
Lowering to `ir` does four things and no more:

1. Resolves every name to a declaration, across imports.
2. Lowers collection and optionality sugar to prelude types.
3. Computes which types satisfy which classes.
4. Resolves target paths and attaches the winning directives.

It does not evaluate constraints, monomorphize generics, or make any decision a backend could reasonably want to make differently.

## Representation

Flat tables with integer IDs, not a pointer graph.

```go
type Model struct {
    Decls []Decl // every declaration, in source order
    Types []Type // type references, interned
}
```

Two tables, two ID spaces.
One declaration table rather than one per kind, because an index has to say which table it indexes and a single space needs no discriminator.
The by-nature split lives inside `Decl`, whose node is a `oneof`.

Every reference is an `ID`, an integer index paired with the node's fully qualified name:

```go
type ID struct {
    Index int
    Name  string // "billing.User.email"
}
```

An index of `-1` means the name did not resolve.
The name is kept regardless, so a diagnostic can say what was written.

The index is what lookups use.
The name is what appears in diagnostics, in ir dumps, and in target paths, and it is stable across edits in a way the index is not.
Carrying both means a dump is readable and a hot loop is still an array index.

Flat tables serialize with no cycle handling and survive the recursion the spec permits in entities and values.

### Declaration shapes

Declarations split by nature rather than collapsing into one tagged message or fragmenting into eight:

- `Struct` covers `entity`, `value`, and `mixin`, which share a shape and differ in meaning. A `StructKind` records which was written.
- `Class` shares that shape and adds what only a contract has.
- `Enum` is its own message: variants, each with optional payload fields.
- `Alias`, `Newtype`, and `Primitive` are their own, being neither structured nor enumerated.

The shared source fidelity, name, doc, position, deprecation, and declaration order, lives in a `Meta` every node carries.

A backend iterating "everything with fields" iterates one shape.
A backend that only handles enums never touches a field that doesn't apply to enums.

### Source fidelity

Every node carries its doc comment, its source position, its deprecation state and message, and its source declaration order.
Doc comments and deprecation are in the spec's contract with backends.
Positions let a backend's own errors point into the `.tdl` file.
Order makes generated output stable and diffs reviewable.

## Sugar

Collection and optionality sugar is lowered.
`[T]` becomes `List<T>`, `{K -> V}` becomes `Map<K, V>`, `T?` becomes `Option<T>`, `T | null` becomes `Nullable<T>`.

The syntactic form is recorded alongside:

```go
type Type struct {
    Ctor  ID  // indexes Decls: List, Option, User
    Args  []ID // indexes Types
    Wrote SyntacticForm // Brackets, Question, Named, ...
}
```

The written form is part of a type's identity for interning.
`[T]` and `List<T>` mean the same type and stay separate entries, because folding them would throw the distinction away the moment a model used both.

A backend that treats all optionality identically ignores `Wrote`.
A Go backend that wants `*T` for `T?` and a wrapper for an explicit `Option<T>` reads it.
The lowering is authoritative; the syntactic form is advisory and never changes what a type *is*.

## Parameters and classes

Type parameters survive.
`Box<T>` reaches a backend as `Box<T>` with `T`'s kind and constraints attached, not as one entry per instantiation found in the model.
A Go or TypeScript backend emits native generics.
A SQL backend monomorphizes on its own terms, which are not the same terms another backend would pick.

Classes, mixins, and declared instances are all present as written.
Alongside them, `ir` ships the computed satisfaction index:

```go
func (m *Model) Satisfying(class ID) []ID
```

A plugin that only needs "which types are `Auditable`" for a class-scoped target directive reads the index and never implements instance resolution.
A backend doing something more involved has the declarations.

## Aliases

Aliases are preserved, and every reference to one carries the resolved underlying type too.

A Go backend emits `type Money decimal` from the alias.
A JSON Schema backend ignores the alias and uses the expansion.
Neither has to walk a chain, and constraint accumulation through newtype chains is resolved before a backend sees it.

## Constraints

Constraints are name plus literal arguments:

```go
type Constraint struct {
    Name string    // "min", "length", "matches"
    Args []Literal
    Pos  Position
}
```

Not a closed set of typed variants.
The spec commits to constraints being syntax that backends interpret, and a closed enum makes every new constraint a change to `ir`, a change to the protobuf, and a version bump for every plugin.

The standard constraints are specified: `min`, `max`, `length`, `matches`, `oneOf`, `unique`, with their argument shapes.
A backend can rely on those.
The compiler checks arity and literal kind against the spec and passes everything through.

## Scope

A backend receives one package.

Declarations from imported packages are not inlined.
A reference into a dependency is an `ID` whose name is qualified with the dependency's package, which a backend either resolves through the model's import table or treats as foreign and maps with a `foreign` directive.

This is what makes separate compilation possible, and it matches what generated code usually wants: an import, not a copy.

## Targets

Resolved directives attach to the nodes they apply to.

By the time a backend runs, the specificity ladder has been applied, dependency and root target blocks have been merged with root winning, class-scoped directives have been expanded across every satisfying type, and equal-specificity conflicts have already been reported as errors.

An `Entity` carries its directives; so does each `Field`.
A backend reads one field on the node in front of it, and never does a lookup or a precedence computation.

The spec's "the model is pure" commitment is about the source language, where directives may not appear in a declaration.
It is not a claim about the compiler's internal representation, where joining them back together is the whole job.

## Wire format

The schema is protobuf, in `proto/`.
The Go types in `ir/` are protoc output, with hand-written methods and index construction in the same package.

Protobuf rather than JSON because plugins are separately compiled programs in unknown languages, on their own release cadence, and field-number compatibility is a real guarantee rather than a convention.

### Handshake

Before the model is sent, `tdl` writes a version message and waits for the plugin to reply.
The plugin either accepts or refuses with a message naming the version it wanted.

A plugin compiled against an older `ir` fails with a sentence a person can act on, rather than silently ignoring the fields it doesn't recognize and emitting subtly wrong code.
Field-number discipline still applies; the handshake is for the cases discipline cannot cover, like a field that changed meaning.

## Placement

```
proto/     # the schema, public API
ir/        # generated Go plus helpers, public API
```

Third parties write in-process backends against `ir`, so it is a supported Go API.
The lowering from `ast` is not: it lives elsewhere and changes freely.

## Deferred

Units.
`unit` declarations and unit expressions are in the grammar and the spec, and `ir` does not model them yet.
A model using units will not lower until it does.
Adding them later is additive: a `Unit` table and a unit-typed argument in `Type.Args`.
