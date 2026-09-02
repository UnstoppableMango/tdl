# Identity as library code

Design document.
Nothing here is built.
It proposes no grammar change and no new keyword; what it moves is where the meaning of `entity` and `value` is written down.

## The problem

The prelude declares identity as library code:

```tdl
/// Anything with identity that survives changes to its contents.
class Entity {
  key
}

/// Anything defined entirely by its contents.
class Value { }
```

No declaration can satisfy either one.
Conformance is nominal and declared, and nothing declares conformance to these.
The keyword decides, and the prelude classes only restate what the keyword already fixed.

[spec.md](../spec.md) then leans on the classes anyway: "`Entity` and `Value` are classes declared by the prelude, so 'any entity' is expressible without a special form."
It is not expressible today.
`requires Entity<T>` parses, resolves, and matches nothing.

One fact, two homes, no link between them.
That is the whole of the problem.

## Why it reads as awkward

Every other abstraction in this language is a prelude declaration with sugar in front of it.
`[T]` is `List<T>`, `{T}` is `Set<T>`, `T?` is `Option<T>`, `T | null` is `Nullable<T>`.
Lowering knows those spellings and nothing about what they mean, which is what makes the prelude replaceable.

Identity is the one concept that took a keyword pair instead, and it is the concept the first design commitment calls first class.
Being first class is why it deserves a keyword; it is not why the meaning has to live in the compiler.

## What is actually load-bearing

Less than the two keywords suggest.
One rule in the compiler branches on entity against value, and it is a legality rule rather than a tag:

- `internal/sema/recursion.go` skips the cycle walk for an entity. Entities may be mutually recursive; a value may reach itself only through a collection or an optional.

Everything else treats them alike.
`internal/sema/satisfy.go` tests for `MIXIN` and never for the other two, and `internal/sema/lower.go` maps the keyword to `ir.StructKind` for backends to read.

So the split carries one rule and one tag, both keyed on the string `"entity"`.

## The shape

The keyword is the conformance.

`entity Order { ... }` conforms to `std.Entity`, and `value Money { ... }` conforms to `std.Value`, the way an `include` confers nothing but a declared `: Auditable` does.
The conformance is written by lowering rather than by the author, since the keyword already said it and saying it twice is what this document is trying to stop.

Three things follow.

`requires Entity<T>` starts meaning what spec.md already claims it means, and a class or a target can ask for "any entity" without the compiler growing a special form.

`checkRecursion` asks whether a declaration satisfies `std.Entity` instead of comparing a keyword string.
That is the same coupling lowering already has to `List` and `Option`: it knows the name and not the meaning.

The prelude classes become the definition rather than a comment.
Replacing the prelude replaces them, which is the property the prelude is supposed to have and does not have here.

## No new keyword

The alternative is a core struct form, say `record`, with `entity` and `value` as sugar over it, matching `[T]` exactly.

This document does not propose that.
Sugar you can write out longhand means `record Order { }` is legal, a struct that is neither an entity nor a value, and the language deliberately makes that choice mandatory.
Adding a third keyword to make two of them derivable also spends more than it saves.

Implicit conformance gets the single source of truth without either cost.
The price is that `entity` and `value` stay in the grammar, so this is not the full uniformity that `List` and `Option` have.
That is the right trade: the keywords are worth keeping for reading, and it is the meaning that needed to move.

## What `class Entity { key }` has to mean

A class body's bare `key` currently reads as "an implementor must have some key."
An entity with no declared key still has identity, supplied by the backend, so under a literal reading such an entity would not satisfy `std.Entity`.

The requirement is therefore "has identity" and not "declares a key field," which is what spec.md says the form means where it introduces it: an implementor must have identity, and which field carries it is the implementor's business.
Read that way `class Entity { key }` needs no change and every entity satisfies it.

Nothing enforces class-body requirements yet, so this is a constraint on the phase that adds enforcement rather than a fix to anything today.

## ir and backends

`ir.StructKind` stays, with `STRUCT_KIND_ENTITY` and `STRUCT_KIND_VALUE` keeping their numbers.

A backend asking "is this an entity" should not have to walk a conformance set, and the field numbers are a compatibility guarantee to plugins.
What changes is that the compiler computes the kind from the conformance rather than from the keyword, which is invisible on the wire.

## What does not change

The grammar. `EntityDecl` and `ValueDecl` keep their productions, and no annotation moves.

The `key` field modifier, and the class-body `key` ambiguity in [backlog.md](../backlog.md), which is a separate defect in what `grammar.ebnf` states and is neither helped nor hurt by this.

`mixin`. A mixin is copied rather than conformed to, so it is a third struct kind for a reason and stays one.

Every `.tdl` file. This is a change to where a meaning is written, not to what anyone writes.

## Open

- Whether an author may also write the conformance by hand, so `entity Order: Entity` is a redundant but legal restatement rather than an error.
- Whether `value X: Entity` is an error or a way to opt a contents-defined type into identity. It should be an error, but stating why needs the answer to the previous question.
- What `Satisfying(std.Value)` returns for an enum. An enum is defined by its contents and has no identity, so it arguably conforms, and nothing in the language says so today.
