# Implementing ir

An implementation plan for [ir.md](ir.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds `ir` and the lowering that produces it.
It does not build backends, the plugin protocol, or the full grammar.

It does include one bite of parser work.
The prelude is written in TDL and parsed like any other input, so lowering cannot run until the parser handles the constructs the prelude uses.
That subset is phase 0.
Everything the prelude does not need, `class`, `mixin`, `instance`, `where`, `unit`, target blocks, belongs to the separate parser plan and arrives before the phases that consume it.

## Layout

```
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

Go unit tests cover the rules whose edges are awkward to express as a whole-file golden: shadowing, recursion, satisfaction with overlapping instances.

`testdata/invalid/` grows lowering cases alongside its parse cases, with the expected diagnostic in `error.golden`.

## Phase 0: parser support for the prelude

The prelude declares primitives, aliases, and parameterized types.
The parser needs `primitive`, `alias`, `TypeParams`, and `Kind` before anything downstream can start.

Done when the prelude source parses cleanly and round-trips through `tdl fmt` unchanged, with conformance cases for each new construct.

## Phase 1: the core schema and the type table

`proto/` gets declarations and type references.
`ir/` gets the generated types, `ID` with its index and qualified name, and the table accessors.

`internal/sema` gains the type table: interning, and lowering of `[T]`, `{T}`, `{K -> V}`, `T?`, and `T | null` to prelude constructors with the syntactic form recorded.

Done when a single-package model of `type` and `enum` declarations lowers to `ir`, every reference is an `ID` resolving to the right table entry, and lowering the same type twice yields the same interned entry.

## Phase 2: name resolution

Scopes, declaration ordering, shadowing, and the recursion the spec permits in entities and values.

Single package only.
An `import` is an error in this phase, which keeps scoping from being designed around a package loader that does not exist.

Done when every name in the conformance corpus resolves to a declaration, unresolved names produce one diagnostic each with a source position, and the whole corpus lowers in one pass.

## Phase 3: `tdl ir`

A command that prints the resolved model as a text tree, matching the shape and conventions of the existing `ast.Dump`, with `--format json` for machine consumption and for plugin authors who want to see what they will receive.

This is the first milestone that is worth showing someone.

Done when `tdl ir` prints every conformance case, the text output is the golden file the corpus checks, and JSON output round-trips through the protobuf.

## Phase 4: the real prelude

Phase 1 lowers sugar to constructors that phase 0's parser can read, but nothing has yet loaded a prelude as a package.

This phase makes the embedded prelude a real compilation unit: parsed, lowered, and merged into the model's scope, with `List`, `Set`, `Map`, `Option`, and `Nullable` as ordinary declarations rather than names lowering knows about.

Done when the sugar lowering in phase 1 resolves through the loaded prelude with no builtin names left in `internal/sema`, and pointing `prelude` at a replacement directory changes what `[T]` means.

## Phase 5: imports

The package loader, `[deps]` prefix resolution, cross-package qualified names, and import cycle detection.

References into another package stay as qualified `ID`s and are not inlined, per the scope decision in ir.md.

Done when a two-package fixture lowers, a reference across the boundary carries the dependency's package in its name, and an import cycle is an error naming the cycle.

## Phase 6: classes, mixins, instances

Requires the parser plan to have delivered `class`, `mixin`, `instance`, `Conforms`, and `where`.

Instance resolution, where-clause constraint checking, functional dependencies, and the computed satisfaction index behind `Model.Satisfying`.

Done when declared instances survive into `ir` unchanged, `Satisfying` answers correctly for the corpus including instances inherited through mixins and conformances, and an unsatisfied `where` constraint is a diagnostic pointing at the use site.

## Phase 7: constraints and deprecation

Constraints as name plus literal arguments, with arity and literal kind checked against the standard set from the spec and everything else passed through.
Doc comments, deprecation state, and declaration order carried onto every node.

Much of this is mechanical and could land earlier; it sits here because it is not on the critical path for anything above it.

Done when every constraint in the corpus reaches `ir` with its arguments and position, a standard constraint with wrong arity is a diagnostic, and an unknown constraint lowers without complaint.

## Phase 8: target resolution

Requires the parser plan to have delivered target blocks.

Path resolution against the model, the specificity ladder, class-path expansion across satisfying types, root-over-dependency merging, and equal-specificity conflicts as errors.

Directives attach to the nodes they apply to, resolved, so no backend performs a lookup.

Done when a target block's directives appear on the right `ir` nodes, a path naming nothing is an error with a position, a class path applies to every satisfying type, and two entries at equal specificity are reported rather than silently ordered.

## After

`tdl check` becomes parse plus full lowering, with `--parse-only` for editors that want the fast path on every keystroke.

At that point `ir` is complete enough for the plugin protocol, which is the next document to implement.

## Not in this plan

- Units. Deferred in ir.md; a model using them will not lower.
- Monomorphization. Parameters stay parameters; a backend that wants concrete types does that itself.
- ir diffing, which incremental generation would need.
