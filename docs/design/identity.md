# Identity as library code

Design document.
Everything outside the open questions is built.

An author declares types in their domain and says which of them have identity.
The language has one keyword for a domain type, and identity is conformance to a prelude class.
`entity`, `value`, and `key` leave the language.

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

Nothing satisfies either class implicitly, and no file in the repository declares conformance to them.
The `entity` and `value` keywords decide instead, and the prelude classes restate what the keywords already fixed.

[spec.md](../spec.md) leans on the classes anyway:

> `Entity` and `Value` are classes declared by the prelude, so "any entity" is expressible without a special form.

`requires Entity<T>` parses, resolves, and matches nothing.
The fact has two homes and no link between them.

The keywords also ask the author a question at the wrong level.
Choosing between `entity` and `value` for every struct, and marking `key` fields, is a decision about storage and equality made while describing the domain.

## Every other abstraction is already library code

`[T]` is `List<T>`, `{T}` is `Set<T>`, `T?` is `Option<T>`, `T | null` is `Nullable<T>`.
Lowering knows those names and nothing about what they mean, which is what makes the prelude replaceable.

Identity is the one concept that took keywords instead.
It stays first class by being a class the compiler knows by name, the same way it knows `List` and `Option`.

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
`type` followed by `:` and a type, with no body, is a newtype, as the grammar already has it.

A type with no identity is the default and is written with nothing extra.
Enums and generic types such as `Option<T>` already work this way.

## Identity is conformance

`: Entity` is the whole statement.
It says the type has identity that survives changes to its contents, and nothing about which fields carry it.

The prelude keeps one class:

```tdl
/// Anything with identity that survives changes to its contents.
class Entity { }
```

`class Value` is deleted.
"Defined entirely by its contents" means "does not conform to `Entity`", and a nominal class cannot express a negation.

`requires Entity<T>` means what spec.md claims it means, and a target block's `Entity => table(snake_case)` applies to every type conforming to it.

## Which fields identify an entity

That is a backend decision, written in a target block when a backend needs it:

```tdl
target sql for shop {
  LineItem => key(order, sku)
}
```

The compiler checks the directive's shape and hands it over, as it does every directive.
A backend that needs no key, or supplies its own, reads nothing.

This removes `key` as a field modifier and as a class-body requirement.
The class-body ambiguity recorded in [backlog.md](../backlog.md) goes with it, since there is no `KeyRequirement` left to collide with `FieldMod`.

The phase in [go-backend-plan.md](go-backend-plan.md) that turns `key` fields into an identity method reads the directive instead.

## Recursion

The recursion rules keep their meaning.
A type conforming to `std.Entity` may be mutually recursive without restriction, since a cycle between entities is a graph of references.
Any other type may reach itself only through a collection or an optional.
So may every enum, including one conforming to `std.Entity`: an enum value is a variant holding its fields inline, and conformance does not make it a reference.

`checkRecursion` asks whether a declaration satisfies `std.Entity` instead of comparing a keyword string.
It walks the AST before `buildSatisfaction` runs (`internal/sema/lower.go`), and conformance can come from an `instance` anywhere in the package, so the check moves after satisfaction is built and reads it.
It reads unconditional conformance only, the set `buildSatisfaction` builds.
A conditional instance, one with parameters or a `requires` clause, stands for a family of types, so it exempts no declaration from the restricted rule; see the last open question below.
The edges it follows, direct, optional, and through a collection, do not change.

## ir and backends

`ir.StructKind` keeps `STRUCT_KIND_ENTITY` and `STRUCT_KIND_VALUE` with their numbers.
A backend asking "is this an entity" should not have to walk a conformance set.
The compiler writes `MIXIN` for a mixin, `ENTITY` for any other struct satisfying `std.Entity`, and `VALUE` for the rest, in that order, so the kind is computed from conformance rather than from a keyword.

`Field.key` (field 3) and `Class.requires_key` (field 6) are removed and their numbers reserved.
That is a breaking change to the plugin protocol and carries the `buf skip breaking` label.
Nothing outside lowering and `ir/dump.go` reads either field.

## Grammar

`ValueDecl` and `EntityDecl` are replaced by a body form of `TypeDecl`:

```ebnf
TypeDecl = "type" identifier [ TypeParams ]
           ( ":" TypeRef [ RequiresClause ] [ ConstraintBlock ]
           | [ Conforms ] [ RequiresClause ] Body ) .
```

A newtype's constraints are written `where { ... }`, so a `{` directly after the colon list or the `requires` clause can only open a body.
One token of lookahead tells the two forms apart.

After the colon, a class means conformance and a type means a newtype's parent.
The reader tells them apart by whether a body follows, which is the price of spending no new keyword.

`FieldMod` loses `"key"`, and `ClassMember` loses `KeyRequirement`.
`entity` and `value` leave the reserved words, so `value` is an ordinary identifier and `Option<T>`'s field needs no special case.

## What changes

Every `.tdl` file that declares a struct.
The rewrite is mechanical: `entity X` becomes `type X: Entity`, `value X` becomes `type X`, and a `key` modifier is dropped.
The repository has 40 such declarations and 15 `key` fields, most of them in `testdata/conformance`.

spec.md's Values, Entities, and Recursion sections, the grammar, the lexer's keyword table, the tree-sitter and TextMate output derived from the grammar, and the goldens.

## What does not change

`enum`, `alias`, and newtypes.

`mixin`.
A mixin is copied rather than conformed to, and it never has identity, so it stays a separate form with `STRUCT_KIND_MIXIN`.

`owned`.
A field whose type conforms to `std.Entity` is a reference, and `owned` still marks composition.

## Open

- Whether an enum may conform to `Entity`.
  Nothing stops it syntactically, and a sum of identities is a meaningful thing for some backends to store.
- Whether an `instance Entity for X` may appear outside the package declaring `X`.
  Conformance from another package would change `X`'s struct kind for one consumer and not another, which argues for requiring it in the declaring package, the way enums are sealed.
- Whether `: Entity` on a generic type applies to every instantiation.
