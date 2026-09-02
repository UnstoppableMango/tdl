# Implementing ir

An implementation plan for [ir.md](ir.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds `ir` and the lowering that produces it.
It does not build backends or the plugin protocol.

The parser is finished.
[parser-plan.md](parser-plan.md) delivered the whole grammar, so every phase below starts from a complete `ast.File` and nothing here is blocked on front-end work.
`prelude/std.tdl` exists and parses; it declares the primitives, collection constructors, `Option`, `Nullable`, the SI base units, and the `Entity` and `Value` classes.

Seven grammar problems surfaced while writing the parser and were fixed in the spec.
Three of them change what lowering receives and are worth stating here:

- The constraint set is open. The compiler checks the arity and argument kinds of the standard names and passes every other name through, so `ir` carries constraints as name plus arguments rather than a closed enum.
- A `<...>` argument is a type or a unit, and a bare name could be either. The parser records `ast.TypeArg` with `Type` set and leaves the decision to kind resolution.
- Directive and constraint arguments are parenthesized, so a directive's arity is unambiguous in the tree.

## Layout

```text
proto/          # ir schema, public
ir/             # generated Go plus helpers, public
internal/sema/  # ast to ir, private
```

`internal/sema` is private and free to change.
`ir` and `proto` are the compatibility surface.

The protobuf is written first for the core, declarations and types, and grows as each phase needs it.
Writing all of it upfront would fix a shape against lowering code that does not exist yet; writing none of it would mean a migration later.

## Errors

Lowering accumulates errors within a pass and stops between passes.

A single run reports every unresolved name in the file, the way the parser reports every syntax error.
It does not then run class satisfaction against a model with unresolved names, because every diagnostic that pass produced would be noise.

No pass ever receives a model that a previous pass rejected.

## Testing

Two layers, both from phase 1 onward.

Golden files extend the existing corpus: `testdata/conformance/*/ir.golden` beside the `source.tdl` already there.
Plain text, so a non-Go implementation can check itself against the same expectations, which is why the corpus is not Go code today.

A case directory holding a `pending` file is skipped, with the file's text as the reason.
The parser rewrite used the marker to keep the suite green while the corpus ran ahead of the implementation, and lowering should use it the same way: write the `ir.golden` a case expects, mark it pending, and delete the marker in the phase that earns it.

Go unit tests cover the rules whose edges are awkward to express as a whole-file golden: shadowing, recursion, satisfaction with overlapping instances.

`testdata/invalid/` grows lowering cases alongside its parse cases, with the expected diagnostic in `error.golden`.

## Phase 1: the core schema and the type table

`proto/` gets declarations and type references.
`ir/` gets the generated types, `ID` with its index and qualified name, and the table accessors.

`internal/sema` gains the type table: interning, and lowering of `[T]`, `{T}`, `{K -> V}`, `T?`, and `T | null` to prelude constructors with the syntactic form recorded.

Done when a single-package model of newtypes, entities, values, and enums lowers to `ir`, every reference is an `ID` resolving to the right table entry, and lowering the same type twice yields the same interned entry.

## Phase 2: name resolution

Scopes, declaration ordering, shadowing, and the recursion the spec permits in entities and values.

Every declaration form the parser produces resolves here: newtypes, aliases, primitives, entities, values, mixins, enums and their variant payloads, classes, instances, and units.
Kind resolution decides what a bare `<...>` argument names, which is the point at which an `ast.TypeArg` becomes a type argument or a unit argument.

Single package only.
An `import` is an error in this phase, which keeps scoping from being designed around a package loader that does not exist.

Done when every name in the conformance corpus resolves to a declaration, unresolved names produce one diagnostic each with a source position, and the whole corpus lowers in one pass.

## Phase 3: `tdl ir`

A command that prints the resolved model as a text tree, matching the shape and conventions of the existing `ast.Dump`, with `--format json` for machine consumption and for plugin authors who want to see what they will receive.

This is the first milestone that is worth showing someone.

Done when `tdl ir` prints every conformance case, the text output is the golden file the corpus checks, and JSON output round-trips through the protobuf.

## Phase 4: the real prelude

Phase 1 lowers sugar to constructors that the parser already reads, but nothing has yet loaded a prelude as a package.

This phase makes `prelude/std.tdl` a real compilation unit: embedded, parsed, lowered, and merged into the model's scope, with `List`, `Set`, `Map`, `Option`, `Nullable`, `Entity`, and `Value` as ordinary declarations rather than names lowering knows about.

Done when the sugar lowering in phase 1 resolves through the loaded prelude with no builtin names left in `internal/sema`, and pointing `prelude` at a replacement directory changes what `[T]` means.

## Phase 5: imports

The package loader, `[deps]` prefix resolution, cross-package qualified names, and import cycle detection.

References into another package stay as qualified `ID`s and are not inlined, per the scope decision in ir.md.

Done when a two-package fixture lowers, a reference across the boundary carries the dependency's package in its name, and an import cycle is an error naming the cycle.

## Phase 6: classes, mixins, instances

Instance resolution, `requires` constraint checking, functional dependencies, and the computed satisfaction index behind `Model.Satisfying`.

`include` expansion belongs here too: a mixin's fields are copied into the including declaration, and the spec's rule that including a mixin is independent of satisfying a class has to survive the copy.

The two instance forms mean the same thing, so lowering normalizes `instance C for T` to `instance C<T>` even though the tree keeps what was written.

The index answers from ground facts only: a declaration that says it conforms, and an instance with concrete arguments, closed over the classes a class requires.
Conditional instances reach `ir` intact and are not expanded.

Done when declared instances survive into `ir`, `Satisfying` answers correctly for the corpus, and an unsatisfied `requires` constraint is a diagnostic pointing at the use site.

## Phase 6b: conditional instance search

`instance <T> Auditable<Page<T>> requires Auditable<T>` says a page of auditable things is auditable, and answering `Satisfying(Auditable)` for `Page<Order>` means matching the head and discharging the condition.

That is a unification engine, and the spec's two termination rules, an instance head applied to distinct parameters and every constraint structurally smaller than the head, are what keep the search finite.
It is scheduled here rather than folded into phase 6 because it is most of a phase on its own, and because everything downstream works without it.

Done when the index answers for an instantiated generic type, a search that would not terminate is rejected at the instance rather than at the use, and the corpus covers a conditional instance that does apply and one that does not.

## Phase 7: constraints, defaults, and deprecation

Constraints as name plus arguments, with arity and argument kinds checked against the standard set from the spec and everything else passed through.
Constraint accumulation down a chain of newtypes, which the spec requires and no backend should have to walk.

Field defaults, including the name form that denotes an enum variant, which is checked against the field's type here rather than by the parser.

Doc comments, deprecation state, and declaration order carried onto every node.

Much of this is mechanical and could land earlier; it sits here because it is not on the critical path for anything above it.

Done when every constraint in the corpus reaches `ir` with its arguments and position, a standard constraint with the wrong arity is a diagnostic, an unknown constraint lowers without complaint, and a default naming a variant the field's type does not have is an error.

## Phase 8: target resolution

Path resolution against the model, the specificity ladder, class-path expansion across satisfying types, and equal-specificity conflicts as errors.

Directives attach to the nodes they apply to, resolved, so no backend performs a lookup.
A directive's name may be a reserved word, since the namespace belongs to the backend; nothing about resolution should assume otherwise.

Root-over-dependency merging is not here.
It needs a dependency's target blocks, and phase 5 decided not to lower dependencies: a qualified reference records the package and stops.
Merging is a phase of its own once there is a reason to load a dependency far enough to read its blocks.

Done when a target block's directives appear on the right `ir` nodes, a path naming nothing is an error with a position, a class path applies to every satisfying type, and two entries at equal specificity are reported rather than silently ordered.

A class path expands across `Satisfying`, which answers about declarations.
It does not expand across `SatisfyingTypes`, so a directive on `Auditable` reaches `Audited` and not the `Page<Audited>` that satisfies the class through a conditional instance.
The model has the answer and target resolution does not read it; closing that is small and belongs with 6b, since both are about the search producing something nothing consumes.

## Phase 8b: dependency target merging

A dependency ships target blocks so its own types appear sensibly in Go without every consumer restating it.
Merging them means loading and lowering the dependency, which nothing else needs, so it waits until something does.

Origin outranks specificity, per [workflow.md](workflow.md): any entry in the root project beats any entry from a dependency, whatever the ladder would otherwise say, and the ladder decides among entries of the same origin.

Done when a dependency's directives reach the root model's nodes, a root entry beats a dependency entry at any specificity, and a conflict between two dependencies is reported.

## Phase 9: units

`ir.md` deferred units and called the addition additive: a `Unit` table and a unit-typed argument in `Type.Args`.
That is what this is, and it is the last thing lowering says it cannot do.

A base unit is the dimension of itself and a derived one reduces to bases, which is what lets the spec's claim that `decimal<N>` and `decimal<kg*m/s^2>` are the same type be an index comparison rather than a walk.

Done when the conformance corpus lowers with no diagnostic at all, and the `deferred` list in `internal/sema/corpus_test.go` is deleted rather than emptied.

Done.
Resolution runs before the rest of lowering and on demand rather than in file order, because a unit may be written after the unit deriving from it, and because a type argument naming a unit needs the answer already computed.
The lowered node is the memo, which is also what makes a cycle reportable at the declaration that closes it.

The `testdata/invalid` corpus was not the place for the new diagnostics.
It is a parse corpus, checked by `parser/conformance_test.go` and by the tree-sitter grammar, and a unit cycle parses cleanly.
They are Go tests in `internal/sema` instead.

## After

`tdl check` becomes parse plus full lowering, with `--parse-only` for editors that want the fast path on every keystroke.

The plugin protocol used to be what came after this, and it arrived first: `docs/design/plugins-plan.md` is complete.

## Not in this plan

- Monomorphization. Parameters stay parameters; a backend that wants concrete types does that itself.
- ir diffing, which incremental generation would need.
- Comment preservation. `tdl fmt` drops ordinary `//` comments today, and fixing it is parser work with no phase in either plan.
