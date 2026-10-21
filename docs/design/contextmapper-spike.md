# Context Mapper spike

Spike report, not a design.
It asks whether TDL should generate [Context Mapper](https://contextmapper.org) Language (CML), and records what a prototype in `backend/contextmapper` found.

## Recommendation

Worth a short design, as an output-only backend in the shape of [likec4-backend.md](likec4-backend.md): a view of the model for people who think in domain-driven design, not a code generator.

- The tactic layer fits well.
  Entities, value objects, enums, newtypes, docs, deprecations, keys, and most constraints have a CML spelling, and aggregates can be inferred from `: Entity` and `owned` with no new syntax.
- The strategic layer is what CML adds and TDL lacks.
  Bounded contexts come from packages and the context map from imports, but relationship roles (`OHS`, `ACL`, customer/supplier) need directives TDL does not have.
- Context Mapper is developed, but its last release is old.
  The repository is active: `master` is 6.12.1-SNAPSHOT on Xtext 2.42 and publishes through the Central Portal.
  The newest release on Maven Central is still `org.contextmapper:context-mapper-dsl` 6.12.0 (August 2024, Xtext 2.32), which is what the spike validates against.
  It is an Xtext and Eclipse stack and ships no command line tool, so checking output means a Java harness over the jar, which the spike builds in `backend/contextmapper/testdata/cmlcheck`.
  The quirks below are 6.12.0's; some, such as `with` failing to lex, may be fixed on `master`.
- Import, CML to TDL, is not worth it now: it needs that Java parser at run time, and CML holds Sculptor's persistence options that TDL has no place for.

## Mapping

| TDL | CML | Notes |
| --- | --- | --- |
| package | `BoundedContext`, named by the package's segments joined with `_` | `context("Name")` on the target block renames it |
| `type X: Entity` | `Entity` | `aggregateRoot` when it roots its aggregate |
| `type X { }` | `ValueObject` | `event` and `command` tags make it a `DomainEvent` or `CommandEvent` |
| mixin | nothing | its fields are already copied into each includer |
| newtype | `BasicType X { <base> value }` | the newtype's constraints sit on `value` |
| alias | nothing | expanded |
| fieldless enum | `enum` | references to it are written `- @Status`, as the validator asks |
| fielded enum | `abstract ValueObject X`, then `ValueObject XVariant extends @X` per variant | |
| field of a primitive | an attribute: `String sku` | |
| field naming a declaration | a reference: `- @Money total` | |
| `[T]`, `{T}` | `List<T>`, `Set<T>` | |
| `T?`, `T \| null` | `nullable` | the two are not told apart |
| `key(...)` | `key` on each field | |
| `min`, `max` | `min = "1"`, `max = "99"` | |
| `length` | `notEmpty` for a lower bound of 1, `length = "N"` or, on a collection, `size = "N"` for the upper | CML takes one number; any other lower bound is lost |
| `matches` | `pattern = "..."` | written as RE2 source; Java reads the common cases alike |
| doc comment, deprecation | the doc string before an object, attribute, or enum value | |
| package doc | a `//` comment before the context | |

Primitives: `string` is `String`, `int` and `int64` `Long`, `int32` `Integer`, `uint32` `Long`, `uint64` `BigInteger`, `float32` `Float`, `float64` `Double`, `bool` `Boolean`, `bytes` `Blob`, `decimal` `BigDecimal`, `date` `Date`, `instant` `Timestamp`.
`uuid` and `duration` are written `UUID` and `Duration`, which CML accepts as Java identifiers.

### Aggregates

CML needs every domain object inside an aggregate (or a module, which its generators ignore), and an aggregate holds at most one `aggregateRoot`.
The prototype infers them:

1. An entity that no other entity owns through an `owned` field roots an aggregate named after it.
1. An owned entity joins its owner's aggregate; a second owner is a `lossy.owned` warning and the first wins.
1. Any other object joins the one aggregate whose entities reach it through fields, without passing through another entity.
1. An object no aggregate reaches joins the one aggregate whose entities it names, so `OrderPlaced { order: Order }` lands beside `Order`.
1. Everything else goes in `Aggregate Shared`, which has no root; the validator accepts that.

This reads `owned` as DDD composition, which is what [identity.md](identity.md) says it means.
A design should add an `aggregate("Name")` directive to override the inference.

## What does not fit

- **Maps.** A CML attribute is `T` or `Collection<T>`; `Map<K, V>` parses only in operation signatures. The field is dropped with `lossy.collection`.
- **Nested collections and optionals**, `[[T]]` or `[T?]`: dropped the same way.
- **Generics.** CML has no type parameters; the declaration is skipped and the cascade follows.
- **Classes.** CML's `Trait` is the nearest thing, but 6.12.0 refuses a `Trait` inside an aggregate and fails to lex `with` on an entity, so a class has no spelling and warns `lossy.class`.
- **Defaults**, `oneOf`, and `unique` have no attribute option and warn `lossy.constraint` or `lossy.default`.
  CML's own `unique` means a unique column, not distinct elements.
- **Units** are refused by `Resolve`, as everywhere else.
- **Services.** CML has `Service` with operations, which could map from the `service` and `rpc` tags the protobuf and likec4 backends read; the prototype does not try.
- **Externs.** A reference into another package is dropped with a warning; see [Context map](#context-map).

## Context map

Probing the validator shows the shape a design would take:

- each package is one `.cml` file holding one `BoundedContext`;
- a model with imports writes `import "./shop_catalog.cml"` and a `ContextMap` that `contains` both contexts with an upstream-downstream relationship, `shop_catalog [U]->[D] shop_billing`;
- a reference into an imported context then validates; without the relationship CML warns that the object is "not reachable through Context Map relationships".

Each downstream file can carry its own map, and a chain of three files validates.
A landscape view of every context still needs one map naming all of them, which a target-block directive or a hand-written file would supply.
Relationship roles (`OHS`, `PL`, `ACL`, `CF`, customer/supplier, partnership, shared kernel) are facts about the import, not the types, and need directives TDL does not have yet.

## CML quirks found

- CML reserves many short and common words as keywords: `email`, `type`, `description`, `key`, `length`, `X`, `x`, `U`, `D`, `S`, `C`, `P`, and about three hundred more, including the literals of its enum rules.
  A name among them is written with Xtext's `^` escape; `keywords.go` is extracted from both grammars.
- An enum attribute must be written as a reference, `- @Status status`, or the validator warns.
- `length`, `size`, `min`, and `max` are strings, but `length` must hold one number.
- Attribute names should start lower case, which TDL fields already do.

## The prototype

`backend/contextmapper` is unregistered: it has no `cmd/tdl-gen-cml`, no registry entry, no `shipped` row, no feature table, and no nix check.
Its tests lower TDL sources, including `testdata/gen/smoke`, and run `cml-check` when it is on `PATH`.

To run them against the real parser:

```shell
cd backend/contextmapper/testdata/cmlcheck
mvn -q -B package dependency:copy-dependencies -DincludeScope=runtime
PATH="$PWD:$PATH" go test ../..
```

Both the smoke model and a shop model with two aggregates, a shared aggregate, an event, and a command validate with no error or warning.

## If a design follows

1. Elements and aggregates: what the prototype does, registered, with `cmd/tdl-gen-cml`, a feature table, and a nix check running the harness through `buildMavenPackage`.
1. Context map: imports, one file per package, and directives for relationship roles.
1. Services: `Service` and operations from the `service` and `rpc` tags.
1. Overrides: `aggregate`, `name`, and the bounded context's `type`, `domainVisionStatement`, and `responsibilities`.

Open questions for that design:

- the target name, `cml` or `contextmapper`;
- whether aggregates are inferred, as here, or always written by hand;
- whether CI validates the output, which costs a Maven closure in the flake, or compares text, as likec4 does.
