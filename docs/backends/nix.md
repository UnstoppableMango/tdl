# Nix

The `nix` backend writes a model as nixpkgs `lib.types` option types: one `<package path>.nix` per model, a function `{ lib }:` returning an attribute set of types, whose values `builtins.toJSON` writes in the JSON wire convention.
Type your own options with them, `lib.mkOption { type = (import ./shop/billing.nix { inherit lib; }).Order; }`, or check a value with `lib.evalModules`.
[#1437](https://github.com/UnstoppableMango/tdl/issues/1437) is its spec, and its sub-issues are the phases; phase 1 is built.

A struct is a submodule, which refuses an attribute it does not declare.
`T?` defaults to null and is left out of the value when unset, so `toJSON` does not write it; `T | null` must be set, and may be null.
A target-block `file` directive names the file in place of the package's path.

## Features

What the backend does with each feature of the language today, measured by `make features` against `testdata/features`.
Yes generates it, Warns loses part of it under the loss codes named, which `tdl.toml` can allow, and No skips it or refuses the model.
Planned names the phase that will generate it.
No importer is planned, so every Import cell is No.

<!-- tdl:features nix -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | No | a `submodule` | Entities, values, and mixins are all submodules, closed to undeclared attributes. |
| Mixin `include` | Yes | No | the mixin's options, copied in | |
| Fieldless enum | Yes | No | `types.enum` of the wire values | A `wire` directive on a variant renames its wire value. |
| Fielded enum | Planned (phase 2) | No | | |
| Newtype | Planned (phase 2) | No | | |
| Alias | Planned (phase 2) | No | | Expanded where used until then. |
| Optional and nullable | Yes | No | `T?` left out of the value when unset, a nullable one a required `nullOr` | |
| List, set, and map | Yes | No | `listOf`, `listOf` refusing duplicates, and `attrsOf` | A map's key is a string, a string's wire form, or a fieldless enum. |
| `min` | Planned (phase 3) | No | | |
| `max` | Planned (phase 3) | No | | |
| `length` | Planned (phase 3) | No | | |
| `matches` | Planned (phase 3) | No | | A pattern POSIX ERE cannot state will warn. |
| `oneOf` | Planned (phase 3) | No | | |
| `unique` | Planned (phase 3) | No | | |
| `owned` | Yes | No | | |
| Field default | Planned (phase 2) | No | | |
| `key` directive | No | No | | The target has no methods. |
| `name` directive | Yes | No | the declaration's attribute | On a field it warns, since `wire` renames a JSON key. |
| `number` directive | No | No | | The target has no field numbers. |
| Doc comment | Yes | No | `#` comments, and a field's `description` | |
| Type parameters | Planned (phase 4) | No | | |
| Class | Warns (`lossy.class`) | No | | Conformance checks nothing at evaluation. |
| Unit | Warns (`lossy.unit`) | No | | A unit-typed field is its base type. |
| Import and extern | Planned (phase 4) | No | | |

<!-- /tdl:features -->
