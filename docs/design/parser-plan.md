# Rewriting the lexer and parser

An implementation plan for bringing the front end up to the grammar in [../grammar.ebnf](../grammar.ebnf).
Phases are ordered by what unblocks [ir-plan.md](ir-plan.md), not by the order the grammar reads.

This is a rewrite, not an extension.
`type` used to introduce a record and now introduces a newtype; records are `entity` and `value`.
The same keyword means something else, annotations are gone, and whitespace handling changes.
The existing corpus, examples, `scratch.tdl`, and the README all stop parsing.

## Breaking changes

No compatibility period and no deprecation warnings.
Nothing depends on the M1 grammar, and carrying two readings of `type` through a rewrite costs more than rewriting the handful of files that use it.

- `type X { ... }` is a syntax error. Records are `entity X { ... }` or `value X { ... }`.
- `@go(...)` annotations are deleted from the lexer, parser, ast, printer, and dump. Target blocks replace them, and land in phase 2 so the gap is short.
- `list<T>` and `map<K, V>` stop parsing. `[T]`, `{T}`, `{K -> V}` are the sugar; `List<T>` and `Map<K, V>` remain valid because they are ordinary prelude types.

## Spec changes

Two grammar problems surfaced while planning, and both are fixed in the spec before any parser code is written.

### Constraint blocks take a `where` prefix

`{` would otherwise open a set type, a declaration body, and a constraint block, and `email: {string} {length 3..254}` needs the parser to tell the second `{` from the first.

```tdl
type Email: string where {
  matches /^[^@]+@[^@]+$/
  length 3..254
}

entity User {
  age: int where { min 0 }
}
```

After a complete TypeRef, `where` and then `{` is a constraint block and nothing else can be.

### Class constraints use `requires`

`where` now belongs to constraints, so constraints on type parameters take their own keyword:

```tdl
value Box<T> requires Ord<T> { ... }
class Sorted<T> requires Ord<T> { ... }
```

`type Money: decimal requires Ord where { min 0 }` reads as two different obligations because they are two different obligations.

### Whitespace is insignificant

Declarations are not newline separated.
The lexer emits no newline tokens and the parser has no separator rules; a declaration or field ends where the next one begins, structurally.

```tdl
enum Role { admin member guest }
```

is legal, and so is the expanded form.
Commas disappear from the grammar rather than remaining as permitted noise.

`tdl fmt` owns layout entirely as a result.
A block stays on one line when it fits within the column limit and expands otherwise, the rule every other formatter uses, and idempotence still holds because the decision depends only on content.

## Keywords

Declaration keywords are reserved: `package`, `import`, `as`, `primitive`, `unit`, `alias`, `type`, `value`, `entity`, `enum`, `class`, `mixin`, `instance`, `target`, `for`, `requires`, `where`, `include`.

Modifiers and constraint names are contextual: `key`, `owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`.
The parser recognizes them by position and they remain ordinary identifiers everywhere else.

A domain model has fields called `key`, `length`, and `owned`, and reserving those words to save a positional check in the parser trades a real cost in the language for a small one in the implementation.

## Scope of the rewrite

Both the lexer and the parser are rewritten.

The lexer changes anyway: whitespace handling, doc comments, regex literals, `->`, `..`, and roughly fifteen new keywords.
The parser's declaration dispatch, type reference parsing, and field parsing are all replaced.
What survives is the shape of the error handling: an `ErrorList` that accumulates rather than aborting, and resynchronization at declaration and field boundaries.

The conformance corpus is the safety net, which is why it is rewritten first.

## Phase 0: spec, grammar, and corpus

Fix `docs/spec.md` and `docs/grammar.ebnf` for the three changes above.

Rewrite `testdata/conformance/`, `testdata/invalid/`, `examples/`, `scratch.tdl`, and the README to the new grammar, all at once, before the parser can read any of it.

The corpus becomes the specification of the target rather than a record of what already works, and tests stay red until phase 1 lands.

Done when the spec and grammar agree with each other, every `.tdl` file in the repository is written in the new grammar, and the corpus covers each construct the later phases add.

## Phase 1: lexer, and the prelude subset

The new lexer: keywords, doc comments, regex literals, the new operators, no newline tokens.

Parser support for exactly what the prelude needs, which is what [ir-plan.md](ir-plan.md) phase 0 is waiting on: `primitive`, `alias`, `TypeParams`, `Kind`, and the bracket collection forms.

Done when the prelude source parses and round-trips through `tdl fmt` unchanged, and `ir-plan.md` is unblocked.

## Phase 2: declarations and target blocks

`entity`, `value`, `enum` with payload variants, `type` as a newtype, `include`, field modifiers, and `deprecated`.

Target blocks land here rather than late, so the window without a way to express backend configuration closes as soon as it opens.

Done when a realistic model, entities with keys and owned relationships, enums with payloads, newtypes, and a target block, parses and formats idempotently.

## Phase 3: constraints

`where { ... }` blocks on newtypes and fields, the six standard constraints, ranges, and regex literals.

Done when every constraint form in the grammar parses with positions, and an unknown constraint name parses as a constraint rather than a syntax error, since backends interpret constraints and the parser does not own the set.

## Phase 4: classes, mixins, instances

`class`, `mixin`, `instance`, `Conforms`, `requires`, associated type requirements and bindings, and functional dependencies.

This unblocks `ir-plan.md` phase 6.

Done when the corpus parses multi-parameter classes with functional dependencies, both `instance C<T>` and the `instance C for T` sugar, and mixins with conformances.

## Phase 5: units

`unit` declarations and unit expressions, with precedence for `*`, `/`, and `^`.

Last because `ir` defers units, so nothing downstream consumes them yet.

Done when unit expressions parse with correct precedence and round-trip through `fmt`.

## Formatting

`ast.Fprint` is rewritten alongside the parser rather than after it.

Every phase's exit criteria include idempotent formatting, because a construct that parses but does not print correctly is not done, and finding that out three phases later means rewriting the printer against a shape that has already spread.

## Not in this plan

- Generic parameters on anything the spec does not yet permit them on.
- Error recovery quality beyond what M1 has. Resynchronization is preserved; making it better is separate work.
