# TDL Specification

TDL is a language for describing domain models.
It says what things are, what identifies them, how they relate, and what values they may hold.
It does not describe behavior, and it has no expressions, control flow, or runtime.

The formal grammar is in [grammar.ebnf](grammar.ebnf).
This document is canonical where the two disagree.

## Design commitments

The language core is small.
Almost everything that looks like a type system is library code written in TDL and shipped in a replaceable prelude.

1. **Identity is first class.** An `entity` has identity that persists across changes to its contents. A `value` is defined entirely by its contents.
2. **The model is pure.** A `.tdl` file describes the domain. Everything a code generator needs lives in a separate `target` block.
3. **Constraints are syntax, not semantics.** The compiler parses and resolves constraints. It does not evaluate or interpret them. Backends decide what a constraint means.
4. **Behavior belongs to backends.** `owned` says a child is part of its parent. It does not say what happens on delete.

## Lexical structure

Identifiers are letters, digits, and underscore, not starting with a digit.
Comments run from `//` to end of line.
Literals are strings (`"..."`), integers, floats, booleans, regexes (`/.../`), and bracketed lists.

Declarations are newline separated.
Commas are permitted but never required as separators inside blocks.

## Packages and imports

```tdl
package billing

import "std/prelude" as _
import "std/si" as si
```

A file declares at most one package.
An import binds a path to a local name; `_` merges the imported names into the current scope without a qualifier.

There is no version syntax.
Versioning a schema is the job of the repository that holds it.

## Roots and the prelude

No type is built in.
A root type is introduced with `primitive`, which tells the compiler only that the type is opaque and irreducible.

```tdl
primitive string
primitive int
primitive bool
primitive bytes
```

The standard prelude declares these roots and the ordinary types built on them (`decimal`, `uuid`, `instant`, `date`, `duration`).
A project may import a different prelude.
The compiler has no opinion about which types exist.

## Types

### Newtypes

`type` declares a distinct type over another type, optionally constrained.

```tdl
type Email: string {
  matches /^[^@]+@[^@]+$/
  length 3..254
}

type UserId: uuid
```

A newtype is not interchangeable with the type it is built on.
`UserId` and `OrderId` are different types even though both are `uuid`.

### Values

A `value` is defined by its contents.
Two values with equal fields are the same value.

```tdl
value Money {
  amount: decimal
  currency: Currency
}
```

### Entities

An `entity` has identity.
Fields marked `key` form its identity; repeating `key` gives a composite identity.

```tdl
entity Order {
  key id: OrderId
  customer: User
  items: [LineItem] owned
  total: Money
}

entity LineItem {
  key order: Order
  key sku: SKU
  quantity: int { min 1 }
}
```

`key` is optional.
An entity without a declared key still has identity; the backend supplies it.

### Enums

An enum is a closed set of variants.
A variant may carry fields, which makes `enum` the language's sum type.
Variants without fields are the degenerate case.

```tdl
enum Payment {
  Card { last4: string, brand: CardBrand }
  Bank { routing: string, account: string }
  Credit
}

enum Currency { USD, EUR, GBP }
```

### Generics

Any declaration may take type parameters.

```tdl
value Page<T> {
  items: [T]
  next: Cursor? | null
}
```

## Type references

### Collections

| Form | Meaning |
| --- | --- |
| `[T]` | ordered, duplicates allowed |
| `{T}` | unordered set |
| `{K -> V}` | map |

Cardinality is not separate syntax.
It falls out of the collection form, optionality, and the `length` constraint: `items: [LineItem] { length 1.. }` is one-or-more.

### Optional and nullable

These are different questions and get different syntax.

| Form | Meaning |
| --- | --- |
| `T` | required, present |
| `T?` | may be absent |
| `T \| null` | present, may be null |
| `T? \| null` | may be absent, and may be null when present |

The distinction matters for partial updates and for formats that can express both, and is preserved through to backends.

### Units of measure

Units are declared, may be derived, and participate in dimensional algebra.

```tdl
unit kg
unit m
unit s
unit N = kg*m/s^2
```

A unit is applied as a type argument.

```tdl
value Weight {
  net: decimal<kg>
  force: decimal<N>
}
```

Unit expressions are normalized to base dimensions before comparison, so `decimal<N>` and `decimal<kg*m/s^2>` are the same type.
`decimal<kg>`, `decimal<m>`, and `decimal` are three different types.

Units may be applied to any type.
The compiler cannot know which types are numeric, because the prelude is replaceable, so it does not try.

`<...>` is a single syntactic form covering both type arguments and unit arguments.
The parser does not distinguish them; the resolver does, against the declaration being applied.

## Relationships

A field whose type is an entity is a reference.
Cardinality comes from the collection form.

`owned` marks composition: the referenced value is part of this one rather than an independent participant.

```tdl
entity Order {
  items: [LineItem] owned   // composition
  customer: User            // reference
  coupon: Coupon?           // optional reference
}
```

Relationships are one-directional as written.
There is no inverse declaration; a backend that needs the reverse direction infers it from the model.

## Constraints

A constraint block may follow a type declaration or a field.
The compiler checks that constraints are well formed and that any names they mention resolve.
It does not check that they are satisfiable, consistent, or meaningful for the type they are attached to.

| Constraint | Form |
| --- | --- |
| `min` | `min 1` |
| `max` | `max 100` |
| `length` | `length 3..254`, `length 1..`, `length 16` |
| `matches` | `matches /^[a-z]+$/` |
| `oneOf` | `oneOf ["a", "b"]` |
| `unique` | `unique` |

The set is closed.
Opening it to arbitrary backend-defined constraints is a later, additive change.

## Targets

Everything a code generator needs lives in a `target` block, never in the model.

```tdl
target go for billing {
  package "github.com/acme/billing"

  User {
    name "Account"
    email => tag "json:\"email_address\""
  }

  Order.items => slice
  Order.tags  => set

  Money   => foreign "github.com/acme/money" "Money"
  decimal => foreign "github.com/shopspring/decimal" "Decimal"
}
```

A target block names a generator and the package it applies to.
Entries are either a path into the model followed by `=>` and a directive, a nested block scoping a path, or a bare directive applying to the enclosing scope.

The compiler resolves every path against the model.
A path that names nothing is an error.
Directives themselves are opaque: the compiler checks their shape and hands them to the backend.

Target blocks may appear in a `.tdl` file or in a separate file.
The standard library ships a target for each supported language, and a project may replace any of them.

## Formatting

`tdl fmt` produces canonical output and is idempotent: formatting canonical output changes nothing.
