# TDL Specification

TDL describes domain models: what things are, what identifies them, how they relate, and what values they may hold.
It has no behavior, expressions, control flow, or runtime.

The formal grammar is in [grammar.ebnf](grammar.ebnf).
This document is canonical where the two disagree.

## Design commitments

The core is small: most of what looks like a type system is TDL library code in a replaceable prelude.

1. **Identity is first class.** A type conforming to the prelude's `Entity` class has identity that persists across changes to its contents. Any other type is defined entirely by its contents.
1. **The model is pure.** A `.tdl` file describes the domain. Everything a code generator needs lives in a separate `target` block.
1. **Constraints are syntax, not semantics.** The compiler parses and resolves constraints but does not evaluate them. Backends decide what a constraint means.
1. **Behavior belongs to backends.** `owned` says a child is part of its parent. It does not say what happens on delete.
1. **Abstraction is library-level.** Generics, kinds, classes, and instances let shared structure be declared once instead of copied between declarations or re-encoded in every backend.

## Lexical structure

Identifiers are letters, digits, and underscore, not starting with a digit.
Declaration keywords are reserved; modifiers and constraint names are not, so `owned`, `length`, and `min` remain usable as field names.
A reserved word followed by `:` is a field name, so a field may be called `type`.
Inside a target block, a directive name or path segment may be a reserved word, since that namespace belongs to the backend.
A package path segment may also be a reserved word, so `package google.type` and `target protobuf for google.type` can mirror another schema's namespace.
Every other name is an ordinary identifier, so `type type { ... }` and `x: type` are both errors.
Comments run from `//` to end of line.
A `///` comment is a doc comment: it attaches to the following declaration, field, or variant and is carried into the model for every target.
Literals are strings (`"..."`), integers, floats, booleans, regexes (`/.../`), and bracketed lists.

Whitespace, including line breaks, is insignificant.
A declaration, field, or variant ends where the next begins, so `enum Role { admin member guest }` is valid on one line.

Commas are required between items inside `<...>`, conformance lists, and list literals, and are not permitted inside `{ ... }` blocks.

## Packages and imports

```tdl
package billing

import "std/prelude" as _
import "std/si" as si
```

A file declares at most one package.
An import binds a path to a local name; `_` merges the imported names into the current scope without a qualifier.

There is no version syntax; the repository holding a schema versions it.

### Visibility

A declaration whose name begins with an upper-case letter is exported from its package.
Everything else is package-private.

Visibility applies to declarations only; a field is visible wherever its declaration is.

`primitive` and `unit` declarations are always exported.
A unit's casing carries meaning (`m` is metre, `M` is mega), so it cannot also signal visibility.

## Roots and the prelude

No type is built in.
`primitive` introduces a root type, telling the compiler only that it is opaque and irreducible.

```tdl
primitive string
primitive int
primitive bool
primitive bytes
primitive int32
primitive uint32
primitive int64
primitive uint64
primitive float32
primitive float64
```

A primitive may take a kind, which is how the collection constructors are introduced.

```tdl
primitive List: type -> type
primitive Set:  type -> type
primitive Map:  type -> type -> type
```

The standard prelude declares these roots and the ordinary types built on them (`decimal`, `uuid`, `instant`, `date`, `duration`).
A project may import a different prelude; the compiler does not decide which types exist.

## Types

### Newtypes

`type` followed by `:` and a type, with no body, declares a newtype: a distinct type over another type, optionally constrained.

```tdl
type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
}

type UserId: uuid
```

A newtype is not interchangeable with the type it is built on: `UserId` and `OrderId` are different types though both are `uuid`.

### Aliases

`alias` declares a transparent abbreviation, expanded before any comparison, so `Handler` and its expansion are the same type.

```tdl
alias Handler = {string -> [Event]}
alias Result<T> = Either<Error, T>
```

Aliases may take parameters, which are applied by substitution.
Use `type` when the distinction should be enforced and `alias` when it should not.

### Values

`type` followed by a body declares a domain type.
One that does not conform to `Entity` is a value, defined by its contents: two values with equal fields are the same value.

```tdl
type Money {
  amount: decimal
  currency: Currency
}
```

A newtype's constraints open with `where`, so a `{` after the name, the conformance list, or the `requires` clause always opens a body.

### Entities

A type conforming to the prelude's `Entity` class is an entity: its identity survives changes to its contents.

```tdl
type Order: Entity {
  id: OrderId
  customer: User
  items: [LineItem] owned
  total: Money
}

type LineItem: Entity {
  order: Order
  sku: SKU
  quantity: int where { min(1) }
}
```

Conformance through an instance, or through a class that requires `Entity`, also makes a type an entity.
A class named `Entity` declared in a file shadows the prelude's, and conforming to it confers no identity.

`: Entity` does not say which fields identify an entity.
That is a backend decision, written in a target block when a backend needs it:

```tdl
target sql for shop {
  LineItem => key(order, sku)
}
```

### Enums

An enum is a closed set of variants.
A variant may carry fields, which makes `enum` the language's sum type.

```tdl
enum Payment {
  Card { last4: string brand: CardBrand }
  Bank { routing: string account: string }
  Credit
}

enum Currency { USD EUR GBP }
```

Enums are sealed: the declaring package fixes the variants and no other package may extend them, so a backend can generate exhaustive handling.

### Recursion

Entities may be mutually recursive without restriction, since a cycle between entities is a graph of references.

```tdl
type Order: Entity { items: [LineItem] owned }
type LineItem: Entity { order: Order }
```

A value or an enum may only reach itself through a collection or an optional, never as a bare field.
An enum holds its variants' fields inline, so conforming to `Entity` does not exempt one.
`type Node { next: Node }` has no finite representation and is an error; `next: Node?` and `children: [Node]` are fine.

Aliases may never be recursive, since they are expanded rather than referenced.

## Parameters and kinds

Any declaration may take parameters.

```tdl
type Page<T> {
  items: [T]
  next: Cursor? | null
}
```

A parameter has a kind: one of the two base kinds, `type` and `unit`, or an arrow between kinds.

| Kind | Inhabited by |
| --- | --- |
| `type` | `Email`, `[LineItem]`, `Money` |
| `unit` | `kg`, `N`, `kg*m/s^2` |
| `type -> type` | `[_]`, `{_}`, `Option` |

Kinds are inferred from how a parameter is used.
A parameter applied to an argument has an arrow kind; one used directly as a field type has kind `type`.

```tdl
type Collection<f, T> {
  items: f<T>          // f is inferred as type -> type
}
```

An explicit annotation is permitted.

```tdl
type Collection<f: type -> type, T: type> {
  items: f<T>
}
```

Kinds also decide what `<...>` means: an argument of kind `unit` attaches a unit, and one of kind `type` fills a declared parameter.
Unit application is therefore not a special case in the grammar.

## Classes, mixins, and instances

Contracts and reuse are separate mechanisms: a class says what a type must provide, and a mixin provides it.

### Classes

A class is a contract and declares nothing into the types that satisfy it.

```tdl
class Auditable {
  createdAt: instant
  updatedAt: instant
}
```

A class may require a field or an associated type.

```tdl
class Paged {
  type Cursor          // an implementor supplies a type
  pageSize: int
}
```

A required field binds every type satisfying the class: the type declares a field of that name and type, itself or through an `include`.
A type that conforms without one is an error where it conforms, so a backend can read a field through the class.

A class may require other classes; satisfying it requires satisfying them.

```tdl
class Auditable: Timestamped { ... }
```

A class may take parameters, including higher-kinded ones, so a target can dispatch on structure instead of on a named type.

```tdl
class Container<f: type -> type> { }

type Page<f, T> requires Container<f> {
  items: f<T>
}
```

A class with more than one parameter states a relationship between types.

```tdl
class Projection<from, to> { }

instance Projection<Order, OrderSummary>
```

A multi-parameter class declares no fields on either participant; backends read the relationship itself.

A functional dependency states that some parameters determine others.

```tdl
class Projection<from, to> | from -> to { }
```

With that dependency, `Projection<Order, OrderSummary>` and `Projection<Order, OrderBrief>` cannot both exist, so `Order` has one projection.

### Constraints on parameters

A `requires` clause constrains the parameters of any declaration that takes them.

```tdl
type Envelope<T> requires Auditable<T> {
  body: T
  receivedAt: instant
}
```

`Entity` is a prelude class, so "any entity" is `requires Entity<T>` without a special form.
A class requiring `Entity` says an implementor must have identity, never which field carries it.

### Mixins

A mixin is reuse: `include` copies its fields into the including declaration.

```tdl
mixin Timestamps {
  createdAt: instant
  updatedAt: instant
}

type User: Entity, Auditable {
  id: UserId
  include Timestamps
  email: Email
}
```

Including a mixin and satisfying a class are independent: a type may satisfy `Auditable` by declaring the fields itself.

### Instances

Conformance is nominal: a type with the right fields does not satisfy a class until it says so, and a type that says so must have them.

Conformance is declared on the declaration:

```tdl
type User: Entity, Auditable { ... }
```

Or separately, which lets a class be applied to a type declared elsewhere:

```tdl
instance Auditable<shipping.Address>
instance Auditable for shipping.Address    // sugar for the same thing
```

The general form is `instance C<T, ...>`.
`for` is sugar available when the class takes exactly one parameter.

An instance supplies bindings for any associated types the class requires.

```tdl
instance Paged for OrderList {
  type Cursor = OrderCursor
}
```

An instance may be parameterized and conditional, which is how generic types participate in classes.

```tdl
class Archived { }

instance <T> Archived<Page<T>> requires Archived<T>
```

That reads: a page of archived things is archived.
An instance supplies no fields, so its head's constructor must declare every field the class requires, as any satisfying type does, and a conditional instance is most useful for a class that requires none.
Two rules keep the search for a conditional instance finite: an instance head must be a type constructor applied to distinct parameters, and every constraint in the `requires` clause must be structurally smaller than the head.
An instance that would require unbounded search is rejected where it is declared, not where it is used.

A separate instance is legal only in the package that declares the class or the package that declares the type, so it is always findable from one end of the relationship.

## Type references

### Collections

| Form | Meaning | Desugars to |
| --- | --- | --- |
| `[T]` | ordered, duplicates allowed | `List<T>` |
| `{T}` | unordered set | `Set<T>` |
| `{K -> V}` | map | `Map<K, V>` |

Collections are prelude types and the bracket forms are sugar, so `List` and `Set` can be passed to a higher-kinded parameter and a replacement prelude can change what a collection is.

Cardinality has no separate syntax.
It comes from the collection form, optionality, and the `length` constraint: `items: [LineItem] where { length(1..) }` is one-or-more.

### Optional and nullable

| Form | Meaning |
| --- | --- |
| `T` | required, present |
| `T?` | may be absent |
| `T \| null` | present, may be null |
| `T? \| null` | may be absent, and may be null when present |

The distinction is preserved through to backends.

Neither is primitive; the prelude declares both, and the syntax is sugar:

```tdl
enum Option<T>   { Some { value: T } None }
enum Nullable<T> { Present { value: T } Null }
```

`T?` is `Option<T>` and `T | null` is `Nullable<T>`.
A replacement prelude may redefine what absence means; the sugar follows whatever `Option` and `Nullable` are bound to.

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
type Weight {
  net: decimal<kg>
  force: decimal<N>
}
```

Unit expressions are normalized to base dimensions before comparison, so `decimal<N>` and `decimal<kg*m/s^2>` are the same type.
`decimal<kg>`, `decimal<m>`, and `decimal` are three different types.

Units may be applied to any type, since with a replaceable prelude the compiler cannot know which types are numeric.

`<...>` is one syntactic form for both type arguments and unit arguments.
The parser does not distinguish them; the resolver does, against the declaration being applied.

## Relationships

A field whose type is an entity is a reference.
Cardinality comes from the collection form.

`owned` marks composition: the referenced value is part of this one rather than an independent participant.

```tdl
type Order: Entity {
  items: [LineItem] owned   // composition
  customer: User            // reference
  coupon: Coupon?           // optional reference
}
```

Relationships are one-directional as written.
There is no inverse declaration; a backend that needs the reverse direction infers it from the model.

## Constraints

A constraint block may follow a type declaration or a field, introduced by `where`.

```tdl
type Email: string where { length(3..254) }

type User: Entity {
  age: int where { min(0) }
}
```

The prefix keeps `{` unambiguous: without it, `email: {string} { length 3..254 }` would open a set type and a constraint block with the same token in the same position.

The compiler checks that constraints are well formed and that any names they mention resolve.
It does not check that they are satisfiable, consistent, or meaningful for the type they are attached to.

A constraint's arguments are parenthesized, and a constraint taking none omits the parentheses.
The set of constraint names is open, so the parser cannot know how many arguments `min` takes, and `min 0 max 100` would be ambiguous.

| Constraint | Form |
| --- | --- |
| `min` | `min(1)` |
| `max` | `max(100)` |
| `length` | `length(3..254)`, `length(1..)`, `length(16)` |
| `matches` | `matches(/^[a-z]+$/)` |
| `oneOf` | `oneOf("a", "b")` |
| `unique` | `unique` |

The compiler checks the arity and argument kinds of the standard names above and passes every other name through to backends untouched.

Constraints accumulate down a chain of newtypes.

```tdl
type Email: string where { length(3..254) }
type WorkEmail: Email where { matches(/@acme\.com$/) }
```

`WorkEmail` carries both constraints.
A newtype narrows its parent and never replaces it, so a value satisfying `WorkEmail` always satisfies `Email`.
The compiler hands the accumulated set to backends without checking that it is satisfiable.

## Defaults

A field may carry a default value.

```tdl
type Order: Entity {
  status: OrderStatus = Pending
  tags: {string} = []
}
```

A default is part of the model rather than a backend setting, because it states what the field means when nothing says otherwise.

A default is a literal or a name.
A name denotes an enum variant, and may be qualified when the reference is ambiguous to a reader.
There are no expressions, so `now` and `uuid()` are backend directives, not defaults.

## Deprecation

`deprecated` marks a declaration, field, or variant, optionally with a reason.

```tdl
deprecated("use billingEmail")
type LegacyContact: Entity { ... }

type User: Entity {
  deprecated email: Email
  billingEmail: Email
}
```

Deprecation is in the language rather than a target, so `tdl check` can report uses of deprecated names and every backend can carry the marker into generated code.

Marking something deprecated changes nothing else; it remains part of the model until it is removed.

## Targets

Everything a code generator needs lives in a `target` block, never in the model.

```tdl
target go for billing {
  package("github.com/acme/billing")

  User {
    name("Account")
    email => tag("json:\"email_address\"")
  }

  Order.items => slice
  Order.tags  => set

  Money   => foreign("github.com/acme/money", "Money")
  decimal => foreign("github.com/shopspring/decimal", "Decimal")
}
```

A target block names a generator and the package it applies to.
Entries are either a path into the model followed by `=>` and a directive, a nested block scoping a path, or a bare directive applying to the enclosing scope.

A path names a declaration, then optionally one of its fields.
For an enum, the second segment names a variant, and a third may name one of that variant's fields, so `Payment.Card => number(4)` and `Payment.Card.last4 => number(2)` each reach one node.

A directive's arguments are parenthesized, and a directive taking none omits the parentheses.
Without the parentheses, `table snake_case` followed by another entry could not be told from `table` applied to three arguments.

A path may name a class, which applies the directive to every type satisfying it.

```tdl
target sql for billing {
  Auditable => trigger("set_updated_at")
  Entity    => table(snake_case)
}
```

When more than one entry could apply to the same thing, the most specific wins.
A directive on a field beats one on its type, which beats one on a class the type satisfies, and a subclass beats a class it requires.
Two entries at the same specificity are an error.
A directive a backend declares repeatable may appear more than once at one specificity, and every entry reaches the backend in source order.

The compiler resolves every path against the model, and a path that names nothing is an error.
Directives are opaque: the compiler checks their shape and hands them to the backend.

Target blocks may appear in a `.tdl` file or in a separate file.
The standard library ships a target for each supported language, and a project may replace any of them.

## Formatting

`tdl fmt` produces canonical output and is idempotent: formatting canonical output changes nothing.

The formatter owns layout, except that a blank line at the top level between two comment groups, or between a comment and the declaration after it, survives.
A block stays on one line when it fits within the column limit and expands to one member per line when it does not.
The decision depends only on content, never on how the input was written.
