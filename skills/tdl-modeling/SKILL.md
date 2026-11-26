---
name: tdl-modeling
description: Write or edit a TDL model, the .tdl files describing a domain's entities, values, enums, newtypes, and constraints. Use when creating or changing a .tdl file, choosing between an entity and a value, adding constraints or defaults, or checking a model with tdl check and tdl fmt.
---

# Modeling in TDL

TDL describes domain models: entities, values, enums, newtypes, classes, and collections.
It has no expressions, control flow, or runtime.
Code is generated from a model by backends, which is another skill's task; this one is the model.

## Before you start

[references/version.md](references/version.md) names the `tdl` release these references were copied from.
Run `tdl version` and compare.
If the versions differ, tell the user that this skill may be out of date for their `tdl`, then carry on: never refuse because of it.

Read [references/spec.md](references/spec.md), the specification, and [references/grammar.ebnf](references/grammar.ebnf), its formal grammar, when a construct here does not cover what you need.
[references/std.tdl](references/std.tdl) is the standard prelude: every type a model can name without declaring it.

## A file

```tdl
package shop

/// An address to send mail to.
type Email: string where { length(3..254) }

type Money {
  amount: decimal
  currency: Currency
}

enum Currency { USD EUR GBP }

/// An order, identified by its id.
type Order: Entity {
  id: uuid
  customer: Email
  items: [LineItem] owned
  total: Money
  note: string?
  status: Status = Draft
}

type LineItem {
  sku: string
  quantity: int where { min(1) }
}

enum Status {
  Draft
  Placed
  Cancelled { reason: string }
}
```

- Whitespace is insignificant and there are no separators: an item ends where the next begins.
  Commas go only inside `<...>`, conformance lists, and list literals, never inside `{ }`.
- A name starting with an upper-case letter is exported from its package.
- `///` is a doc comment, carried into every target; `//` is an ordinary comment.

## Choosing a declaration

- `type Name { ... }` is a value: two with equal fields are the same.
- `type Name: Entity { ... }` is an entity: its identity survives changes to its contents.
  Use it for things that are tracked over time, such as an order or a user.
  Which fields identify it is a target's `key` directive, not part of the model.
- `type Name: Base` with no body is a newtype: a distinct type over `Base`, which may add constraints.
  Two newtypes over `uuid` are not interchangeable, so prefer one per identifier.
- `alias Name = T` is a transparent abbreviation, never distinct from `T`.
- `enum` is a closed set of variants, each of which may carry fields, so it is also the sum type.
- `mixin` holds fields an `include` copies into a type.
- `class` is a contract a type conforms to, as `Entity` is; `instance` declares a type's conformance separately.

## Types

- `[T]` is a list, `{T}` a set, `{K -> V}` a map.
- `T?` is optional, may be absent; `T | null` is present but may be null.
- A field whose type is an entity is a reference; `owned` makes it composition.
- A type is finite: `type Node { next: Node }` is an error, while `next: Node?` and `children: [Node]` are fine.
  Entities may refer to each other in cycles.

## Constraints and defaults

A constraint block follows a type declaration or a field, introduced by `where`.
The standard constraints are `min(1)`, `max(100)`, `length(3..254)` (also `length(1..)` and `length(16)`), `matches(/^[a-z]+$/)`, `oneOf("a", "b")`, and `unique`.
Constraints accumulate down a chain of newtypes: a newtype narrows its base, never replaces it.

A default is a literal or an enum variant: `status: Status = Draft`, `tags: {string} = []`.
There are no expressions, so a timestamp of now is a target directive, not a default.

Mark something going away with `deprecated`, optionally with a reason: `deprecated("use billingEmail") email: Email`.

## Keeping the model about the domain

A fact true of the domain, such as a constraint, a default, or a deprecation, goes in the model.
How one language spells, lays out, or numbers the model goes in a `target` block, and the model never names a target.
Do not add a field or a type only to satisfy one language.

## Checking your work

Run both after every edit, and fix what they report before you finish:

- `tdl check model.tdl` parses and lowers the file, and prints every error with its position.
- `tdl fmt -w model.tdl` rewrites it in canonical form; `tdl fmt --check` lists files that are not.

An error naming an undefined type usually means a missing declaration or import, or a misspelled prelude name; [references/std.tdl](references/std.tdl) lists those.
