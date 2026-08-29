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
5. **Abstraction is library-level.** Generics, kinds, classes, and instances exist so that shared structure is declared once and reused, rather than copied between declarations or re-encoded in every backend.

## Lexical structure

Identifiers are letters, digits, and underscore, not starting with a digit.
Comments run from `//` to end of line.
A comment beginning `///` is a doc comment: it attaches to the declaration, field, or variant that follows, is carried through to the model, and is available to every target.
Literals are strings (`"..."`), integers, floats, booleans, regexes (`/.../`), and bracketed lists.

Whitespace is insignificant.
A declaration, field, or variant ends where the next one begins, so line breaks carry no meaning and `enum Role { admin member guest }` is as valid as the expanded form.

Commas separate items inside `<...>`, conformance lists, and list literals, where they are required.
They are not separators inside `{ ... }` blocks and are not permitted there.

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

### Visibility

A declaration whose name begins with an upper-case letter is exported from its package.
Everything else is package-private.

Visibility is a property of declarations only.
Fields are always visible wherever their declaration is, because a field that no backend can see is not something a schema can usefully say.

`primitive` and `unit` declarations are always exported.
Units are exempt because their casing carries meaning: `m` and `M` are metre and mega, and forcing a unit to capitalize would change what it denotes.

## Roots and the prelude

No type is built in.
A root type is introduced with `primitive`, which tells the compiler only that the type is opaque and irreducible.

```tdl
primitive string
primitive int
primitive bool
primitive bytes
```

A primitive may take a kind, which is how the collection constructors are introduced.

```tdl
primitive List: type -> type
primitive Set:  type -> type
primitive Map:  type -> type -> type
```

The standard prelude declares these roots and the ordinary types built on them (`decimal`, `uuid`, `instant`, `date`, `duration`).
A project may import a different prelude.
The compiler has no opinion about which types exist.

## Types

### Newtypes

`type` declares a distinct type over another type, optionally constrained.

```tdl
type Email: string where {
  matches /^[^@]+@[^@]+$/
  length 3..254
}

type UserId: uuid
```

A newtype is not interchangeable with the type it is built on.
`UserId` and `OrderId` are different types even though both are `uuid`.

### Aliases

`alias` declares a transparent abbreviation.
An alias is not a new type; it is expanded before any comparison, so `Handler` and its expansion are the same type.

```tdl
alias Handler = {string -> [Event]}
alias Result<T> = Either<Error, T>
```

Aliases may take parameters, which are applied by substitution.
Recursive aliases are an error.

Use `type` when the distinction should be enforced and `alias` when it should not.

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
  quantity: int where { min 1 }
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
  Card { last4: string brand: CardBrand }
  Bank { routing: string account: string }
  Credit
}

enum Currency { USD EUR GBP }
```

Enums are sealed.
The set of variants is fixed by the declaring package, and no other package may extend it.
This is what allows a backend to generate exhaustive handling.

### Recursion

Entities may be mutually recursive without restriction.
A cycle between entities is a graph of references, which every backend can represent.

```tdl
entity Order  { items: [LineItem] owned }
entity LineItem { order: Order }
```

A value may only reach itself through a collection or an optional, never as a bare field.
`value Node { next: Node }` is an error because it has no finite representation; `next: Node?` and `children: [Node]` are both fine.

Aliases may never be recursive, since they are expanded rather than referenced.

## Parameters and kinds

Any declaration may take parameters.

```tdl
value Page<T> {
  items: [T]
  next: Cursor? | null
}
```

A parameter has a kind.
There are two base kinds, `type` and `unit`, and arrows between them.

| Kind | Inhabited by |
| --- | --- |
| `type` | `Email`, `[LineItem]`, `Money` |
| `unit` | `kg`, `N`, `kg*m/s^2` |
| `type -> type` | `[_]`, `{_}`, `Option` |

Kinds are inferred from how a parameter is used.
A parameter applied to an argument has an arrow kind; one used directly as a field type has kind `type`.

```tdl
value Collection<f, T> {
  items: f<T>          // f is inferred as type -> type
}
```

An explicit annotation is permitted, and is worth writing when a parameter is never applied or when the inferred kind would be surprising.

```tdl
value Collection<f: type -> type, T: type> {
  items: f<T>
}
```

Kinds also decide what `<...>` means.
An argument of kind `unit` attaches a unit; an argument of kind `type` fills a declared parameter.
This is why unit application is not a special case in the grammar.

## Classes, mixins, and instances

Contracts and reuse are separate mechanisms, because they solve different problems.
A class says what a type must provide.
A mixin provides it.

### Classes

A class is a contract.
It declares nothing into the types that satisfy it.

```tdl
class Auditable {
  createdAt: instant
  updatedAt: instant
}
```

A class may require a field, a key, or an associated type.

```tdl
class Tenanted {
  key                  // an implementor must have some key
  tenant: TenantId
}

class Paged {
  type Cursor          // an implementor supplies a type
  pageSize: int
}
```

A class may require other classes, which makes satisfying it require satisfying them.

```tdl
class Auditable: Timestamped { ... }
```

A class may take parameters, including higher-kinded ones, which lets a target dispatch on structure rather than on a named type.

```tdl
class Container<f: type -> type> { }

value Page<f, T> requires Container<f> {
  items: f<T>
}
```

A class may take more than one parameter, which states a relationship between types rather than a property of one.

```tdl
class Projection<from, to> { }

instance Projection<Order, OrderSummary>
```

A multi-parameter class declares no fields on either participant.
Its content is the relationship itself, which backends read.

A functional dependency states that some parameters determine others, which makes the relationship a function rather than a table.

```tdl
class Projection<from, to> | from -> to { }
```

With that dependency, `Projection<Order, OrderSummary>` and `Projection<Order, OrderBrief>` cannot both exist, so a backend asking for "the projection of `Order`" always gets one answer.

### Constraints on parameters

A `requires` clause constrains parameters.
It applies to any declaration that takes parameters.

```tdl
value Envelope<T> requires Auditable<T> {
  body: T
  receivedAt: instant
}
```

`Entity` and `Value` are classes declared by the prelude, so "any entity" is expressible without a special form.

### Mixins

A mixin is reuse.
`include` copies its fields into the including declaration.

```tdl
mixin Timestamps {
  createdAt: instant
  updatedAt: instant
}

entity User: Auditable {
  key id: UserId
  include Timestamps
  email: Email
}
```

Including a mixin is how a type usually comes to satisfy a class, but the two are independent: a type may satisfy `Auditable` by declaring the fields itself.

### Instances

Conformance is nominal, never inferred from shape.
A type that happens to have the right fields does not satisfy a class until it says so.

Conformance is declared in one of two places.
On the declaration:

```tdl
entity User: Auditable { ... }
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

An instance may itself be parameterized and conditional, which is how generic types participate in classes.

```tdl
instance <T> Auditable<Page<T>> requires Auditable<T>
```

That reads: a page of auditable things is auditable.
Resolving a conditional instance is a search, so two rules keep it finite.
An instance head must be a type constructor applied to distinct parameters, and every constraint in the `requires` clause must be structurally smaller than the head.
An instance that would require unbounded search is rejected at the point of declaration rather than at use.

A separate instance is legal only in the package that declares the class or the package that declares the type.
Instances belonging to neither are rejected, so an instance is always findable from one end of the relationship.

## Type references

### Collections

| Form | Meaning | Desugars to |
| --- | --- | --- |
| `[T]` | ordered, duplicates allowed | `List<T>` |
| `{T}` | unordered set | `Set<T>` |
| `{K -> V}` | map | `Map<K, V>` |

Collections are prelude types, not built-ins, and the bracket forms are sugar.
This is what allows `List` and `Set` to be passed to a higher-kinded parameter, and what allows a replacement prelude to change what a collection is.

Cardinality is not separate syntax.
It falls out of the collection form, optionality, and the `length` constraint: `items: [LineItem] where { length 1.. }` is one-or-more.

### Optional and nullable

These are different questions and get different syntax.

| Form | Meaning |
| --- | --- |
| `T` | required, present |
| `T?` | may be absent |
| `T \| null` | present, may be null |
| `T? \| null` | may be absent, and may be null when present |

The distinction matters for partial updates and for formats that can express both, and is preserved through to backends.

Neither is primitive.
The prelude declares them as ordinary types, and the syntax is sugar:

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

A constraint block may follow a type declaration or a field, introduced by `where`.

```tdl
type Email: string where { length 3..254 }

entity User {
  age: int where { min 0 }
}
```

The prefix is what keeps `{` unambiguous.
Without it, `email: {string} { length 3..254 }` would open a set type and a constraint block with the same token in the same position.

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

Constraints accumulate down a chain of newtypes.

```tdl
type Email: string where { length 3..254 }
type WorkEmail: Email where { matches /@acme\.com$/ }
```

`WorkEmail` carries both constraints.
A newtype narrows its parent and never replaces it, so a value satisfying `WorkEmail` always satisfies `Email`.
The compiler collects the accumulated set and hands it to backends; it does not check that the set is satisfiable, since it does not interpret constraints.

## Defaults

A field may carry a default value.

```tdl
entity Order {
  status: OrderStatus = Pending
  tags: {string} = []
}
```

A default is part of the model rather than a backend setting, because it states something about the domain: what this field means when nothing said otherwise.
A default must be a literal; there are no expressions, so `now` and `uuid()` are not defaults but backend directives.

## Deprecation

`deprecated` marks a declaration, field, or variant as on its way out, optionally with a reason.

```tdl
deprecated("use billingEmail")
entity LegacyContact { ... }

entity User {
  deprecated email: Email
  billingEmail: Email
}
```

Deprecation is in the language rather than in a target, so `tdl check` can report uses of deprecated names and every backend can carry the marker into generated code without being told how.

Marking something deprecated changes nothing else.
It remains part of the model until it is removed.

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

A path may name a class, which applies the directive to every type satisfying it.
This is the main practical payoff of classes: a rule is written once rather than repeated per type.

```tdl
target sql for billing {
  Auditable => trigger "set_updated_at"
  Entity    => table snake_case
}
```

When more than one entry could apply to the same thing, the most specific wins.
A directive on a field beats one on its type, which beats one on a class the type satisfies, and a subclass beats a class it requires.
Two entries at the same specificity are an error rather than a silent choice.

The compiler resolves every path against the model.
A path that names nothing is an error.
Directives themselves are opaque: the compiler checks their shape and hands them to the backend.

Target blocks may appear in a `.tdl` file or in a separate file.
The standard library ships a target for each supported language, and a project may replace any of them.

## Formatting

`tdl fmt` produces canonical output and is idempotent: formatting canonical output changes nothing.

Because whitespace is insignificant, the formatter owns layout entirely.
A block stays on one line when it fits within the column limit and expands to one member per line when it does not.
The decision depends only on content, never on how the input was written, which is what makes it idempotent.
