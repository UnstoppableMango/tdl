# Anonymous unions

Design document.
Nothing here is built; [unions-plan.md](unions-plan.md) orders the work.

A field wants `Card | Bank` or `string | int` without declaring an enum for it.
`T | null` is already sugar for the prelude enum `Nullable<T>`; a union is the same move one step further.

```tdl
type Order {
  payment: Card | Bank
  reference: string | int
  note: (string | Markdown)?
  parent: Order | OrderId | null
}
```

## What a union is

`A | B` is an anonymous enum.
Lowering synthesizes an enum with one variant per alternative, and each variant holds the alternative in one field named `value`, as `Nullable`'s `Present` does.
Every backend that writes a fielded enum already writes a union's shape; a backend whose target has native unions recognizes the synthesized enum and writes the native form instead.

There is no prelude `Union` and no variadic kind.
The kind system is unchanged, and so is what a replacement prelude may redefine: the sugar for a union is the enum itself, not a name lowering looks up.

### A union is a set

`A | B` and `B | A` are the same type.
Duplicates collapse, a union inside a union flattens, and an alias whose right side is a union flattens where it is used, so `alias Funds = Card | Bank` makes `Funds | Credit` the union of three.
One union is interned per distinct set of alternatives, so two fields spelling the same set share one entry in `Types` and one synthesized declaration.
Alternatives are compared after aliases expand: `alias Id = string` makes `Id | string` the one-alternative union, which is an error (see [Rules](#rules)).

The synthesized enum's variants are sorted by their names, so the order a source spells is never observable.

### Optional and nullable

`?` binds to one type, so an optional union is parenthesized: `(A | B)?` is `Option<A | B>`.
`A | B?` and `A? | B` are errors that suggest the parenthesized form.

`null` may only be the last alternative, and it wraps the rest: `A | B | null` is `Nullable<A | B>`, and `(A | B)? | null` is `Nullable<Option<A | B>>`, matching the order `T? | null` has today.
`null` is never a variant.

## Names

A union has no declared name, but every nominal target needs one: Go, Rust, Haskell, C#, Java, and the schema IDLs all declare the sum type somewhere.

The synthesized declaration is named by joining its variants' names, sorted, with `Or`: `Card | Bank` is `BankOrCard`, and `string | int` is `IntOrString`.
A `name` directive on a field holding the union renames the declaration for that target; two fields naming one union differently is an error, since they share it.

A variant is named for its alternative:

| Alternative | Variant |
| --- | --- |
| a declaration `Card` | `Card` |
| a primitive `string` | `String` |
| `[Item]` | `ItemList` |
| `{Item}` | `ItemSet` |
| `{string -> int}` | `StringIntMap` |
| `Page<Item>` | `ItemPage` |

These are the names Smithy's backend already gives a collection's shape.
Two alternatives producing one variant name, such as `billing.Card | legacy.Card`, is an error.

A synthesized name colliding with a declared one is an error at the union, naming the declaration it collides with.

## The wire

The JSON wire convention ([wire.go](../../backend/internal/emit/wire.go)) gains one rule: a union tags only what needs a tag.

- An alternative whose value is a JSON object, a struct or a fielded enum, carries the discriminant beside its own properties, valued with the alternative's name, as a fielded enum's variant does.
- Every other alternative is written bare: a string, number, boolean, or array is its own value, and a map is its own object.

```json
{ "payment": { "kind": "Card", "last4": "4242", "brand": "visa" }, "reference": 1042 }
```

A fielded enum as an alternative keeps its own type: its value sits under `value` beside the union's discriminant, so `Payment | Refund` reads `{"kind": "Payment", "value": {"kind": "Card", ...}}` and neither tag shadows the other.

The discriminant is the target block's `discriminant` directive, else `kind`, as for a fielded enum.
A struct alternative with a field of that name is an error at the union.

### Distinguishable alternatives

A reader takes a bare value's JSON type to choose its alternative, so lowering rejects a union in which two alternatives have the same JSON shape and are not both tagged:

| Shape | Alternatives |
| --- | --- |
| string | `string`, `bytes`, `decimal`, `uuid`, `instant`, `date`, `duration`, a fieldless enum |
| number | `int`, `int32`, `int64`, `uint32`, `uint64`, `float32`, `float64` |
| boolean | `bool` |
| array | a list or a set |
| object, bare | a map |
| object, tagged | a struct, a fielded enum |

A newtype has its base's shape.
`string | int` and `Card | Bank | [Card]` lower; `string | uuid`, `int | float64`, `Email | string`, and `{string -> int} | Card` do not.
The table is the wire convention's, so it lives beside it in one place both `sema` and `emit` read.

A target with no JSON reader still sees the same union: protobuf, Thrift, and Smithy write tagged unions natively, and the rule costs them nothing.

## Rules

- A union has at least two alternatives after flattening, aliases, and duplicates; `string | string` is an error.
- An alternative is a type: a declaration, a primitive, a collection, or an applied generic. A class is not a type, and neither is a unit.
- A type parameter is not an alternative. Its JSON shape is unknown until it is applied, so `T | string` cannot be checked where it is written. Lifting this means checking each instantiation; see [Deferred](#deferred).
- A union is an enum to the recursion rules: it may reach itself only through a collection or an optional. `type Tree { children: [Tree | Leaf] }` is fine; `type Node { next: Node | Leaf }` is not.
- A constraint on a union-typed field is an error. A constrained alternative is a newtype: `string where { length(1..) } | int` is written `NonEmpty | int`.
- A union conforms to no class and has no instances, since nothing can name it. A class field typed with a union is satisfied by a field holding the same set.
- Target-block paths cannot name a union. The synthesized declaration takes directives only from the fields that hold it, and only `name`.

## Grammar

```ebnf
TypeRef   = Alternative { "|" Alternative } [ "|" "null" ] .
Alternative = CoreType [ "?" ] .
CoreType  = ListType | SetOrMapType | ParenType | NamedType .
ParenType = "(" TypeRef ")" .
```

The grammar admits `A? | B` so the parser can report it with the fix; lowering accepts `?` only on a lone alternative.
Inside `<...>`, `(` also opens a unit group (`decimal<(kg*m)>`): a group holding `|`, `?`, or a collection is a type, one holding `*`, `/`, or `^` is a unit, and a bare name stays a type reference resolved by kind, as it is today.
`|` already separates functional dependencies in a class header, which follows type parameters and never a `TypeRef`.

## ir

```protobuf
enum SyntacticForm {
  SYNTACTIC_FORM_UNION = 7; // A | B
}

message Enum {
  bool union = 5; // synthesized from A | B; variants each hold one field named value
}
```

The synthesized declaration is an ordinary `Decl` holding an `Enum`, in the package of the file that first spells the union, positioned there.
A `Type` naming it sets `ctor` to it and `wrote` to `SYNTACTIC_FORM_UNION`; it has no arguments, since no alternative is a parameter.
`ir.Dump` prints the reference as written and the declaration under its synthesized name, marked `union`.

A backend that does nothing new writes a union as the fielded enum it is, which compiles everywhere fielded enums do.
A model with a union is therefore never a protocol break for an older plugin; it only reads less natively.

`internal/unlower` writes no declaration for a union enum and prints each reference as `A | B`, so `tdl import` and the round-trip corpus read back what was written.

## Targets

The design's preference holds: encode the union in the target's type system wherever it has one, and generate no validator.

- **TypeScript**: a native union. A struct alternative is `Card & { kind: "Card" }`, a fielded enum alternative is `{ kind: "Payment"; value: Payment }`, and every other is its type, so `Card | Bank | string` is `(Card & { kind: "Card" }) | (Bank & { kind: "Bank" }) | string`.
- **JSON Schema and OpenAPI 3.x**: a `oneOf`. A struct alternative is an `allOf` of its `$ref` and a `const` discriminant, and a fielded enum alternative is an object of a `const` discriminant and a `value` holding its `$ref`; when every alternative is tagged, OpenAPI also writes a `discriminator`. OpenAPI 2.0 has no `oneOf`, so a union warns and its field is skipped, as a fielded enum's is.
- **Go**: a sealed interface named for the union, with one unexported marker method. The union is synthesized in the model's own package, so a struct or enum the model declares implements the marker directly (a fielded enum through each variant's struct); a primitive, a collection, or an extern, which another Go package declares and this one cannot add a method to, is wrapped in a named type (`type IntOrStringString string`). Under a `json` directive the interface's holder marshals by the wire rules above.
- **Haskell**: a sum type, one constructor per variant, prefixed with the union's name since constructors share a namespace (`IntOrStringInt Int`).
- **protobuf**: a message holding a `oneof` of one field per alternative, the shape a fielded enum's variants already take.
- **Thrift**: a `union` of one field per alternative.
- **Smithy**: a `union` shape of one member per alternative.
- **GraphQL**: a union of object types. A struct alternative is its own type; a fielded enum, itself a GraphQL union and so not a member of another, and every bare alternative are wrapped in an object type with one field, `value`, as a fieldless variant is today.
- **Salesforce**: no column holds a union, so a field holding one warns, as a fielded enum's does.
- **LikeC4**: no element; a union is drawn as the field's relationships to each alternative once relationships exist.

Backends designed and not yet built take the same line: Rust an `enum` with a newtype variant per alternative, C# and Java a sealed hierarchy as for a fielded enum, Python `A | B` with `Annotated` tagging for pydantic, and OCaml and F# a sum type.
Each design names the mapping when its phase for enums is planned.

### Import

A target's native union reads back as a union, not a declared enum, wherever the reader can tell:

- TypeScript `A | B` whose members are named types, primitives, or arrays, and a `T & { kind: "T" }` member as a tagged struct.
- A Smithy or Thrift union whose members each hold one alternative under a name the variant table produces, and a GraphQL union of object types, are read as a union when the declaration's name is the joined name; otherwise they stay a declared enum.
- Go and Haskell read a union back under `roundtrip` only, from the annotation the generator writes.

A union a reader cannot name stays a declared enum, which is still the same model.

## Language server

Go to definition on an alternative jumps to it, as on any type name, and hover on `|` prints the synthesized declaration's name and variants.
A completion after `|` offers types, as after `:`.

## Deferred

- **Type parameters as alternatives.** `T | U` needs every instantiation checked for distinguishable alternatives where it is applied, which `sema` does not track.
- **A per-union discriminant.** Target paths cannot name a union, so `discriminant` is per block; a `discriminant` directive on the field would need the same agreement rule `name` has.
- **Structural alternatives.** An inline struct as an alternative wants anonymous structs first.
- **Literal alternatives.** `"a" | "b"` is `oneOf("a", "b")` on a string today; a literal type is a different feature.
