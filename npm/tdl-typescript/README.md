# @unmango/tdl-typescript

A TypeScript content mapper for the [Type Description Language](https://github.com/UnstoppableMango/tdl).
With it, a TypeScript project imports a `.tdl` model directly, and `tsc` checks the importing code against it and reports a TDL error in the `.tdl` file:

```ts
import type { Order, OrderStatus } from "./shop.tdl";
```

It needs TypeScript 7.1 or later, which added content mappers.
Name the package in `tsconfig.json`:

```json
{
  "compilerOptions": {},
  "contentMappers": [{ "package": "@unmango/tdl-typescript", "extensions": [".tdl"] }]
}
```

and run `tsc --runExternalCode`, without which `tsc` runs no mapper.
The module each `.tdl` file maps to is the one `tdl gen --target typescript` writes for it.

The package carries no JavaScript and no binary.
`tsc` runs `tdl-gen-typescript content-mapper` from the package's directory, so `tdl-gen-typescript` must be on `PATH`: from the Nix flake's `tdl` package, or from a [release tarball](https://github.com/UnstoppableMango/tdl/releases), which holds it beside `tdl`.

See [Importing TDL into TypeScript](https://github.com/UnstoppableMango/tdl/blob/main/docs/architecture/typescript-content-mapper.md) for how the mapper works.
