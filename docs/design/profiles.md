# Profiles

Design document.

A profile is a named, reusable set of target entries for one backend.
It states conventions once, such as "every value is omitted" or "every entity gets a key", and a target block applies it by name.
The spec's [Targets](../spec.md#targets) section says the standard library ships a target for each supported language and a project may replace any of them; profiles are that mechanism.

Profiles are resolved in `sema`.
A backend receives the same resolved directives it receives from a target block, so every backend supports profiles with no code of its own.
The `likec4` backend is the first to ship profiles.

## Declaring a profile

```tdl
package std.likec4

profile services for likec4 {
  @*      => element(none)
  @entity => element(entity)
  views(false)
}
```

`profile <name> for <target> { ... }` is a top-level declaration in any `.tdl` file and belongs to that file's package, so the profile above is `std.likec4.services`.
Its body is target entries, written as in a target block.
`profile` is a declaration keyword and so reserved, and like every reserved word it is a field name when followed by `:`.

A profile names no package, so its paths cannot name the declarations of the model it is applied to.
A path in a profile is one of:

- a kind selector, described below;
- a class, from the prelude or imported by the profile's file, which expands across every declaration satisfying it;
- a prelude type, such as `decimal => foreign("github.com/shopspring/decimal", "Decimal")`.

A path with a field segment is an error, since a profile knows no fields.
A bare directive applies to the block, as it does in a target block.

## Kind selectors

A kind selector matches declarations by what they are rather than by name:

| Selector | Matches |
| --- | --- |
| `@entity` | a structure whose `StructKind` is `ENTITY` |
| `@value` | a structure whose `StructKind` is `VALUE` |
| `@mixin` | a structure whose `StructKind` is `MIXIN` |
| `@enum` | an enum, with or without variant fields |
| `@newtype` | a newtype |
| `@class` | a class |
| `@*` | every declaration in the model's own files |

The `@` keeps a selector apart from a declaration named `value` or `entity`.
Kind selectors are valid in a target block as well as in a profile.
A selector matches only the model's own declarations, never the prelude's.

Kind selectors extend the specificity ladder in the spec.
From most to least specific: a field, a type, a subclass, a class it requires, a kind selector, `@*`.

## Applying a profile

```tdl
target likec4 for shop with std.likec4.services {
  WidgetService => element(service)
}
```

`with` follows the block's package and names one profile by qualified name.
The name resolves like a type name: a profile in the same package, one in a file the block's file imports, or a shipped one under `std`.
A profile whose `for` names a different target is an error at the name.

A block applies at most one profile.
A profile may apply another with `with` in its own header, so conventions compose as a chain.
A cycle is an error at the `with` that closes it.

`with` is contextual: it is recognized only between a target block's package and its `{`, so it remains a valid name everywhere else.

## Precedence

A block and the profiles it applies are layers.
The block is the top layer, then its profile, then the profile that one applies, and so on.

Every entry in a higher layer beats every entry in a lower layer for the same directive on the same node, whatever their specificity.
The block therefore has the last word: `WidgetService => element(service)` above beats the profile's `@* => element(none)` because it is in the block, and would beat it as a type path anyway.
Within one layer the ordinary ladder applies, and two entries at the same specificity in one layer are an error.

A directive the backend declares repeatable reaches it from every layer, lowest layer first, then in source order.

## What a backend receives

Directives arrive on the nodes they resolve to, as they do from a target block.
The IR carries two fields for profiles, each on a number no other field has used, so a plugin built without them reads the model unchanged:

- `ir.Directive.from_profile` (field 6): the qualified name of the profile an entry came from, beside `from_class`, so a warning can say where a directive was written.
- `ir.TargetBlock.profile` (field 4): the qualified name of the profile the block applies.

A backend that wants different behavior from a profile reads the directives it already declares.
It never branches on a profile's name.

## Shipped profiles

Shipped profiles are `.tdl` files embedded with `go:embed` beside the prelude, one per target, at `prelude/std/<target>.tdl` in package `std.<target>`.
`sema` loads one only when a block names a profile under `std.<target>`, so a model that applies none pays nothing.

A project replaces a shipped profile by declaring its own, in its own package, and naming that one.
A shipped profile is never edited in place by a project.

A test lowers every shipped file, so a profile that names a directive or class that does not resolve fails CI.

## Examples

Each profile below sets directives its backend already reads.

```tdl
package std.protobuf

profile editions for protobuf {
  edition("2024")
}
```

```tdl
package std.go

profile keyed for go {
  Entity => key(id)
}
```

A profile carries fixed conventions.
A value that differs per project, such as the Salesforce `prefix` or a Go `package` path, stays in the block.

## Open questions

- **Parameters.** `with std.salesforce.namespaced("acme")` would let a profile carry a per-project value; it would need a syntax for referring to the argument inside the body.
- **Several profiles per block.** `with a, b` needs an order between `a` and `b`; a chain covers the same need with an explicit order.
- **Block-scope `out`.** A profile could set a conventional `out`; whether a profile may set where files are written is unsettled.
