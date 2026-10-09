# Implementing ir

An implementation plan for [ir.md](ir.md).
Phases are ordered by dependency, and each states what makes it done.

Phases 1 through 9 are done, and the conformance corpus lowers with no diagnostic.
Phase 8b is partial.

## Scope

This plan builds `ir` and the lowering that produces it, starting from a complete `ast.File`.
Backends and the plugin protocol are out of scope.

Three spec decisions shape what lowering receives:

- The constraint set is open. The compiler checks the arity and argument kinds of the standard names and passes every other name through, so `ir` carries a constraint as a name plus arguments.
- A `<...>` argument is a type or a unit, and a bare name could be either. The parser records `ast.TypeArg` with `Type` set and leaves the decision to kind resolution.
- Directive and constraint arguments are parenthesized, so a directive's arity is unambiguous in the tree.

## Layout

```text
proto/          # ir schema, public
ir/             # generated Go plus helpers, public
internal/sema/  # ast to ir, private
```

`ir` and `proto` are the compatibility surface; `internal/sema` is free to change.
The protobuf grows as each phase needs it.

## Errors

Lowering accumulates errors within a pass and stops between passes.
A run reports every unresolved name in a file, and no pass receives a model a previous pass rejected, since its diagnostics would be noise.

## Testing

Golden files: `testdata/conformance/*/ir.golden` beside each `source.tdl`, plain text so a non-Go implementation can check itself against them.
A case directory holding a `pending` file is skipped, with the file's text as the reason, and the phase that earns it deletes the marker.

Go unit tests in `internal/sema` cover rules whose edges are awkward as a whole-file golden: shadowing, recursion, overlapping instances, and diagnostics for files that parse cleanly, such as a unit cycle.
`testdata/invalid/` is a parse corpus, also checked by the tree-sitter grammar, so a lowering diagnostic on a file that parses belongs in a Go test.

## Phase 1: the core schema and the type table (done)

Declarations and type references in `proto/`, `ID` and the table accessors in `ir/`, and interning in `internal/sema`, with `[T]`, `{T}`, `{K -> V}`, `T?`, and `T | null` lowered to prelude constructors and the syntactic form recorded.

Done when a single-package model of newtypes, entities, values, and enums lowers, every reference is an `ID` resolving to the right entry, and lowering the same type twice yields the same entry.

## Phase 2: name resolution (done)

Scopes, declaration ordering, shadowing, and the recursion the spec permits in entities and values, for every declaration form.
Kind resolution decides whether a bare `<...>` argument is a type or a unit.

Done when every name in the corpus resolves, each unresolved name is one positioned diagnostic, and the corpus lowers in one pass.

## Phase 3: `tdl ir` (done)

A command printing the resolved model as a text tree in `ast.Dump`'s conventions, with `--format json` for plugin authors.

Done when `tdl ir` prints every conformance case, the text output is the golden file, and the JSON round-trips through the protobuf.

## Phase 4: the real prelude (done)

`prelude/std.tdl` is embedded, parsed, lowered, and merged into the model's scope, with `List`, `Set`, `Map`, `Option`, `Nullable`, and `Entity` as ordinary declarations whose names lowering knows and whose meaning it does not.

Done when sugar resolves through the loaded prelude with no builtin names left in `internal/sema`, and a replacement prelude changes what `[T]` means.

## Phase 5: imports (done)

The package loader, `[deps]` prefix resolution, cross-package qualified names, and import cycle detection.
References into another package stay qualified `ID`s, per the scope decision in [ir.md](ir.md).

Done when a two-package fixture lowers, a cross-package reference carries the dependency's package in its name, and an import cycle is an error naming the cycle.

## Phase 6: classes, mixins, instances (done)

Instance resolution, `requires` checking, functional dependencies, and the satisfaction index behind `Model.Satisfying`, answered from ground facts.
`include` copies a mixin's fields into the including declaration, independent of class satisfaction.
`instance C for T` is normalized to `instance C<T>`.

Done when declared instances survive into `ir`, `Satisfying` answers correctly for the corpus, and an unsatisfied `requires` is a diagnostic at the use site.

## Phase 6b: conditional instance search (done)

Answering whether `Page<Order>` satisfies `Archived` through `instance <T> Archived<Page<T>> requires Archived<T>` means matching the head and discharging the condition.
The spec's termination rules, an instance head applied to distinct parameters and every constraint structurally smaller than the head, keep the search finite.

Done when the index answers for an instantiated generic type, a search that would not terminate is rejected at the instance, and the corpus covers a conditional instance that applies and one that does not.

## Phase 7: constraints, defaults, and deprecation (done)

Constraints with arity and argument kinds checked for the standard names, and accumulated down newtype chains.
Field defaults, including a name denoting an enum variant, checked against the field's type.
Doc comments, deprecation, and declaration order on every node.

Done when every corpus constraint reaches `ir` with its arguments and position, a standard constraint with the wrong arity is a diagnostic, an unknown one lowers silently, and a default naming a variant the field's type lacks is an error.

## Phase 8: target resolution (done)

Path resolution, the specificity ladder, class-path expansion across satisfying declarations, and keeping every candidate at equal specificity.
Lowering does not judge a tie: `gen.CheckDirectives` reports one as an error unless the backend declared the directive repeatable.
A directive's name may be a reserved word, since that namespace belongs to the backend.

Done when directives appear on the right nodes, a path naming nothing is a positioned error, a class path applies to every satisfying type, and two entries at equal specificity are reported.

A class path does not expand across `SatisfyingTypes`, so a directive on `Auditable` reaches `Audited` and not a `Page<Audited>` that satisfies it through a conditional instance.
Closing that is small and belongs with 6b.

## Phase 8b: dependency target merging (partial)

A dependency ships target blocks so its types generate sensibly without every consumer restating them.
Origin outranks specificity, per [workflow.md](workflow.md): a root entry beats any dependency entry, and the ladder decides among entries of one origin.

Done when a dependency's directives reach the root model's nodes, a root entry beats a dependency entry at any specificity, and a conflict between two dependencies is reported.

Block-scope directives are done: each `ir.Import` carries the bare top-level directives of its dependency's target blocks for the dependency's package, read from the parse tree without lowering the dependency, and `ir.Dump` prints them.
The protobuf backend reads them to import an extern from the file its dependency generates.
Declaration-level directives reach the extern naming the declaration: a dependency's `Decl => d(...)` entry, and the bare directives of a `Decl { ... }` block, read from the parse tree the same way.
A root entry naming the extern beats the dependency's, whatever its specificity.
An entry reaching beneath a declaration is not carried, since an extern is referred to only as a whole.
A transitive dependency's entries, and a conflict between two dependencies, are not done.

## Phase 9: units (done)

A `Unit` table and a unit-typed argument in `Type.Args`, with a derived unit reduced to bases so `decimal<N>` and `decimal<kg*m/s^2>` compare by index.
Resolution runs before the rest of lowering and on demand, because a unit may be written after the unit deriving from it and a type argument naming a unit needs its reduction already.
The lowered node is the memo, which also makes a cycle reportable at the declaration that closes it.

Done when the conformance corpus lowers with no diagnostic.

## After

`tdl check` parses only.
It should run full lowering, with `--parse-only` for editors that want the fast path.

## Not in this plan

- Monomorphization. A backend that wants concrete types does it itself.
- ir diffing, which incremental generation would need.
