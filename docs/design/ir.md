# The ir package

Design document.
[ir-plan.md](ir-plan.md) tracks the implementation.

`ir` is the resolved semantic model: what `ast` becomes after names are resolved, sugar is lowered, and instances are checked.
It is the only thing a backend sees, whether it imports the Go package in process or reads a protobuf from stdin as a plugin.

## What resolution does

`ast` mirrors source one-to-one and leaves every name unresolved.
Lowering to `ir` does four things:

1. Resolves every name to a declaration, across imports.
1. Lowers collection and optionality sugar to prelude types.
1. Computes which types satisfy which classes.
1. Resolves target paths and attaches the winning directives.

It does not evaluate constraints, monomorphize generics, or make any decision a backend could reasonably want to make differently.

## Representation

Flat tables with integer IDs, not a pointer graph.

```go
type Model struct {
    Decls []Decl // every declaration, in source order
    Types []Type // type references, interned
    Units []Unit // quantities, interned on their reduced dimensions
    Externs []Extern // declarations in dependencies, see Scope
}
```

Each table is its own ID space.
There is one declaration table rather than one per kind, so an index needs no discriminator; `Decl`'s node is a `oneof`.

Every reference is an `ID`, an integer index paired with the node's fully qualified name:

```go
type ID struct {
    Index int
    Name  string // "billing.User.email"
}
```

An index of `-1` means the name did not resolve, and the name is kept so a diagnostic can say what was written.
Lookups use the index; diagnostics, dumps, and target paths use the name, which is stable across edits where the index is not.

Flat tables serialize with no cycle handling and survive the recursion the spec permits in entities and values.

## Units

A unit is a quantity reduced to base dimensions: a sorted list of base units and their exponents, with cancelled dimensions dropped.
A base unit is the dimension of itself, and a derived one reduces to the bases it was written over.

The spec says `decimal<N>` and `decimal<kg*m/s^2>` are the same type, so a unit is interned on its dimensions alone, and every spelling of one quantity reaches one entry.

`UnitDef` is the declaration the source wrote; `Unit` is the quantity, and two declarations of the same quantity share one.
The unit records the first spelling that reached it, so a dump can print `kg/m/s^2 (N/m^2)`, but the spelling takes no part in identity.

A unit argument is an entry in the type table with `unit` set where a named type sets `ctor`, so it is an ordinary entry in `Type.Args`.
The parser does not tell type and unit arguments apart inside `<...>`; lowering does, against the declaration each name resolves to.

### Declaration shapes

Declarations split by nature:

- `Struct` covers a `type` with a body and a `mixin`. A `StructKind` records which: `MIXIN` for a mixin, `ENTITY` for a type satisfying `std.Entity`, and `VALUE` for the rest.
- `Class` shares that shape and adds what only a contract has.
- `Enum` holds variants, each with optional payload fields and the directives a target block attached to it.
- `Alias`, `Newtype`, and `Primitive` are their own, being neither structured nor enumerated.

### Source fidelity

Every node carries a `Meta` with its name, doc comment, source position, deprecation state and message, and declaration order.
Positions let a backend's errors point into the `.tdl` file, and order keeps generated output stable.

## Sugar

`[T]` becomes `List<T>`, `{K -> V}` becomes `Map<K, V>`, `T?` becomes `Option<T>`, and `T | null` becomes `Nullable<T>`.

The syntactic form is recorded alongside:

```go
type Type struct {
    Ctor  ID  // indexes Decls: List, Option, User
    Args  []ID // indexes Types
    Wrote SyntacticForm // Brackets, Question, Named, ...
}
```

A type parameter, an extern, or a unit is set instead of `Ctor`.

The written form is part of a type's interning key, so `[T]` and `List<T>` are separate entries for the same type.
The lowering is authoritative and the form is advisory: a backend that distinguishes spellings reads `Wrote`, and every backend here ignores it.

## Parameters and classes

Type parameters survive: `Box<T>` reaches a backend with `T`'s kind and constraints attached, not as one entry per instantiation.
A language with generics emits them natively, and a backend that monomorphizes does so on its own terms.

Classes, mixins, and declared instances are all present as written, alongside the computed satisfaction index:

```go
func (m *Model) Satisfying(class ID) []ID
```

A backend that only needs "which types are `Auditable`" reads the index and never implements instance resolution.

The index has two halves:

- `Satisfying` answers about declarations, from ground facts: a declaration that says it conforms, and an instance with concrete arguments, closed over the classes a class requires.
- `SatisfyingTypes` answers about instantiated types, from the conditional instance search. Given `instance <T> Auditable<Page<T>> requires Auditable<T>`, `Page` satisfies nothing on its own and `Page<Order>` is a type rather than a declaration. The search matches an instance head against a type and discharges the conditions, and the spec's two rules on an instance are checked where it is written so the search terminates.

Neither half lists a dependency's declaration, since those are not in the declaration table.

## Aliases

An alias is preserved as a declaration, and a reference to it names the alias; the alias's `target` is its expansion.
`emit.Resolve` expands aliases for a backend that wants the underlying type.
Constraints accumulated through a newtype chain are resolved before a backend sees them, in `Newtype.value_constraints`.

## Constraints

Constraints are a name plus literal arguments:

```go
type Constraint struct {
    Name string    // "min", "length", "matches"
    Args []Literal
    Pos  Position
}
```

The set is open: the spec treats constraints as syntax that backends interpret, and a closed enum would make every new constraint a change to `ir`, the protobuf, and every plugin.

The standard constraints, `min`, `max`, `length`, `matches`, `oneOf`, and `unique`, are specified with their argument shapes.
The compiler checks their arity and literal kinds and passes every other name through.

## Scope

A backend receives one package.

Declarations from imported packages are not inlined.
A reference into a dependency is an `ID` into an `externs` table, one entry per foreign declaration the model mentions, carrying the declaring package and the name.
A backend resolves it through the model's import table or maps it with a `foreign` directive.
A target path naming a declaration a `_` import merged in resolves to its `Extern` entry, whose `directives` field holds the winning directives; the path reaches nothing beneath it.

Whether the dependency declares that name is not checked, since that would mean resolving every reachable package to lower one file.

This allows separate compilation, and generated code usually wants an import rather than a copy.

## Targets

Resolved directives attach to the nodes they apply to.

By the time a backend runs, the specificity ladder has been applied and class-scoped directives have been expanded across every satisfying declaration.
Entries tied at one specificity are checked against the directives the backend declares repeatable: a tie on any other directive is an error before the backend runs, and a repeatable one reaches it as every entry in source order.

An `Import` carries its dependency's block-scope directives: the bare directives at the top level of each of the dependency's target blocks for its own package, each naming the block's target.
A backend generating a reference into the dependency reads from them where the dependency's declarations are generated, such as a protobuf `file` or `package`.
They are read from the dependency's parse tree, so none of the dependency is lowered.
Merging a dependency's declaration-level directives is not done; see [ir-plan.md](ir-plan.md) phase 8b.

A `Decl`, a `Field`, an enum variant, and an `Extern` each carry their directives, so a backend never does a lookup or a precedence computation.

The spec's "the model is pure" commitment is about the source language, where directives may not appear in a declaration; joining them back together is the compiler's job.

## Wire format

The schema is protobuf, in `proto/`, and the Go types in `ir/` are protoc output with hand-written methods beside them.
Plugins are separately compiled programs in unknown languages, so field-number compatibility is a real guarantee where a JSON convention would not be.
[plugins.md](plugins.md) describes the handshake that refuses a model a plugin cannot read.

## Placement

```text
proto/     # the schema, public API
ir/        # generated Go plus helpers, public API
```

Third parties write in-process backends against `ir`, so it is a supported Go API.
The lowering from `ast` lives in `internal/sema` and changes freely.
