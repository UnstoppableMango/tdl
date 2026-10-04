# Identity as library code

Design document.
Everything outside the open questions is built.

An author declares domain types with one keyword, `type`, and says which have identity by conforming to the prelude class `Entity`.
There are no `entity`, `value`, or `key` keywords.

## Why

`[T]`, `{T}`, `T?`, and `T | null` are sugar for prelude types; lowering knows the names and nothing about their meaning, which keeps the prelude replaceable.
Identity follows the same pattern: a class the compiler knows by name, the way it knows `List` and `Option`.

Keywords gave identity two homes, the keyword and the prelude class, with no link between them, so `requires Entity<T>` matched nothing.
They also forced a storage and equality decision on every struct while describing the domain.

## What an author writes

```tdl
type Money {
  amount: decimal
  currency: Currency
}

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

`type` followed by a body declares a domain type.
`type` followed by `:` and a type, with no body, is a newtype.
A type with no identity needs nothing extra.

## Identity is conformance

`: Entity` says the type has identity that survives changes to its contents, and nothing about which fields carry it.

The prelude declares one class:

```tdl
/// Anything with identity that survives changes to its contents.
class Entity { }
```

There is no `Value` class: "defined entirely by its contents" means "does not conform to `Entity`", and a nominal class cannot express a negation.

`requires Entity<T>` matches every conforming type, and a target block's `Entity => table(snake_case)` applies to each.

## Which fields identify an entity

A backend decision, written in a target block when a backend needs it:

```tdl
target sql for shop {
  LineItem => key(order, sku)
}
```

The compiler checks the directive's shape and passes it on.
The Go backend reads it to generate `Key()` ([go-backend-plan.md](go-backend-plan.md)).

## Recursion

A type conforming to `std.Entity` may be mutually recursive without restriction, since a cycle between entities is a graph of references.
Any other type may reach itself only through a collection or an optional.
So may every enum, including one conforming to `std.Entity`, because an enum value holds its variant's fields inline.

`checkRecursion` (`internal/sema/recursion.go`) runs after `buildSatisfaction` and reads unconditional conformance only.
A conditional instance (one with parameters or a `requires` clause) stands for a family of types, so it exempts no declaration.

## ir and backends

`ir.StructKind` keeps `STRUCT_KIND_ENTITY` and `STRUCT_KIND_VALUE`, so a backend need not walk a conformance set.
Lowering writes `MIXIN` for a mixin, `ENTITY` for any other struct satisfying `std.Entity`, and `VALUE` for the rest.

`Field.key` (field 3) and `Class.requires_key` (field 6) are removed and their numbers reserved.

## Grammar

```ebnf
TypeDecl = "type" identifier [ TypeParams ]
           ( ":" TypeRef [ RequiresClause ] [ ConstraintBlock ]
           | [ Conforms ] [ RequiresClause ] Body ) .
```

A newtype's constraints are written `where { ... }`, so a `{` directly after the colon list or `requires` clause opens a body, and one token of lookahead separates the forms.
After the colon, a class means conformance and a type means a newtype's parent; the reader tells them apart by whether a body follows.

`entity` and `value` are not reserved, so `value` is an ordinary identifier.

## Unchanged

- `enum`, `alias`, and newtypes.
- `mixin`: copied rather than conformed to, never has identity, and lowers to `STRUCT_KIND_MIXIN`.
- `owned`: a field whose type conforms to `std.Entity` is a reference, and `owned` marks composition.

## Open

- Whether an enum may conform to `Entity`. Nothing stops it syntactically, and a sum of identities is meaningful for some backends to store.
- Whether `instance Entity for X` may appear outside the package declaring `X`. Conformance from another package would change `X`'s struct kind for one consumer and not another, which argues for requiring the declaring package, as enums are sealed.
- Whether `: Entity` on a generic type applies to every instantiation.
