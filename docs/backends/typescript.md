# TypeScript

The `typescript` backend writes one `.ts` file per model of JSON wire types: interfaces, both enum shapes, and newtypes as aliases.
It writes no runtime code unless a `construct` or `brand` directive asks for [constructors](#constructors).
[schema-backends.md](../architecture/schema-backends.md) is its design.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Import generates the feature and reads it back, and counts only what reading back loses, so a loss Generate already shows is not counted again.

<!-- tdl:features typescript -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | Warns (`lossy.struct-kind`) | `interface` | Entities, values, and mixins are all interfaces, so each reads back as a value. |
| Mixin `include` | Yes | Warns (`lossy.include`) | the mixin's properties, copied in | The include is flattened into the fields it copies. |
| Fieldless enum | Yes | Yes | a union of string literals | |
| Fielded enum | Yes | Yes | a union of interfaces discriminated on `kind` | A `discriminant` directive renames `kind`. |
| Newtype | Yes | Yes | `type N = Base` | A plain alias, so parsed JSON needs no cast; under a `brand` directive, an alias intersected with a brand keyed by its qualified name, so two newtypes over one base are not interchangeable. Under `construct` or `brand`, a function of the same name builds one. |
| Alias | Yes | Warns (`lossy.alias`) | expanded where used | |
| Optional and nullable | Yes | Yes | `name?: T`, or a union with `null` | |
| List, set, and map | Warns (`lossy.collection`) | Yes | `T[]` and `Record<K, V>` | A set is an array, and reads back as a list. |
| `min` | Warns (`lossy.constraint`) | Yes | | Warns unless `min` and `max` together bound at most 16 integers, which are written as a literal union, or a newtype's constructor checks it. |
| `max` | Warns (`lossy.constraint`) | Yes | | Warns unless `min` and `max` together bound at most 16 integers, which are written as a literal union, or a newtype's constructor checks it. |
| `length` | Warns (`lossy.constraint`) | Yes | | A list's exact length of at most 16 is a tuple; any other length warns, since TypeScript has no type for it, unless a newtype's constructor checks it. |
| `matches` | Warns (`lossy.constraint`) | Yes | | TypeScript has no type for a pattern; a newtype's constructor checks it under `patterns`. |
| `oneOf` | Yes | Yes | a union of literal types | |
| `unique` | Warns (`lossy.constraint`) | Yes | | A set or a list is an array, which does not enforce uniqueness; a newtype's constructor checks it on scalar elements. |
| `owned` | Warns (`lossy.owned`) | Yes | | TypeScript has no ownership, and a referenced type is embedded by value. |
| Field default | Warns (`lossy.default`) | Yes | | The default is not written. |
| `key` directive | No | No | | The output is types with no runtime code, so there is no key function to write. |
| `name` directive | Yes | Warns (`lossy.name`) | the property or type name | The new name reads back as the field's own name rather than a directive. On a fielded variant it renames the interface, never the `kind` value; on a fieldless variant it has no effect and warns. |
| `number` directive | No | No | | JSON has no field numbers. |
| Doc comment | Yes | Yes | a JSDoc comment | |
| Type parameters | No | No | | Type parameters are not generated yet, so a parameterized declaration and every one applying it are skipped. |
| Class | Warns (`lossy.class`) | Yes | | A class is not generated yet, and a conformance to one is dropped. |
| Unit | Warns (`lossy.unit`) | Yes | | A unit is not generated yet, and a quantity is written as its primitive. |
| Import and extern | Yes | Yes | `import type` | From a `foreign` module, or from the dependency's file when both target blocks set `out`. |

<!-- /tdl:features -->

## Brands

A newtype is a plain alias by default, so two newtypes over one base are interchangeable.
The `brand` directive keeps them apart: on the target block it brands every newtype, and on a newtype it brands that one.

```tdl
type Secret: string
type Username: string

target typescript for auth {
  Secret => brand
  Username => brand
}
```

```ts
export type Secret = string & { readonly __brand: { readonly "auth.Secret": true } };
export type Username = string & { readonly __brand: { readonly "auth.Username": true } };
```

tsc then refuses a `Secret` where a `Username` is expected, and a plain `string` for either; a value is made with the newtype's [constructor](#constructors), which `brand` turns on, `Username("ann")`.
A newtype over a branded newtype adds its own brand to its parent's, so it is still its parent.
`zod`'s brands are not these, so a project using both backends picks one as the source of nominal types.

## Constructors

The `construct` directive writes a function per newtype with the newtype's own name, beside its type, so a call site makes a value without a cast: on the target block for every newtype, or on one newtype.
`brand` turns it on wherever it is written, and `construct` anywhere but a newtype warns.

```tdl
type Secret: string
type Code: string where { length(2..8) }
type Grade: string where { oneOf("gold", "silver") }

target typescript for auth {
  construct
}
```

```ts
export type Secret = string;
export const Secret = (v: string): Secret => v as Secret;

export type Code = string;
export const Code = (v: string): Code => {
  if ([...v].length < 2 || [...v].length > 8) {
    throw new RangeError("Code: length(2..8)");
  }
  return v as Code;
};

export type Grade = "gold" | "silver";
export const Grade = (v: "gold" | "silver"): Grade => v as Grade;
```

A constructor takes the newtype's base as the backend writes it, so a newtype over a newtype takes its parent, and checks only what that type cannot state.
A constraint written as a literal union or a tuple is the parameter's type and has no check.
Each other constraint, its own and inherited, throws a `RangeError` naming the newtype and the constraint: `min` and `max` on a number, `length` on a string (in code points), a list, a set, or a map, and `unique` on a list or a set of strings, numbers, booleans, or fieldless enums.
`matches` is checked only under the `patterns` directive on the target block, as in `zod`, and a pattern whose shape JavaScript's engine backtracks on keeps warning.
A constraint a constructor cannot check warns `lossy.constraint` with the reason.
A checked constraint does not read back: import reads a constructor as a `construct` directive and warns that its checks are not read.
