# Importing TDL into TypeScript

Design document.
[typescript-content-mapper-plan.md](typescript-content-mapper-plan.md) orders the work and says which phases are built.

`tdl gen` writes a `.ts` file, and a TypeScript project imports what was written.
The types are stale from the moment a `.tdl` file changes until someone runs `tdl gen` again.

TypeScript 7.1 adds content mappers: a package that turns a file of another language into a module for `tsc` to check.
[Motherload](https://github.com/colinhacks/motherload) uses them to import TOML, YAML, and CSV as typed modules.
A content mapper for TDL lets a TypeScript file import a model directly:

```ts
import type { Order, OrderStatus } from "./shop.tdl";
```

`tsc` asks the mapper for the module, checks the importing file against it, and reports a TDL error in the `.tdl` file at its line and column.
An editor running TypeScript 7.1 does the same on every keystroke.

The mapper is an alternative a project opts into.
`tdl gen`, `tdl gen --verify`, and the Nix flake module stay the hermetic path.

## Content mappers

This section records the protocol as TypeScript 7.1.0-beta and Motherload 0.2.0 implement it.
[microsoft/typescript-go#4712](https://github.com/microsoft/typescript-go/pull/4712) introduced it and is the reference; where this section says a field is inferred, phase 2 checks it there first.

A project names a mapper package and the extensions it owns in `tsconfig.json`, and runs `tsc` with `--runExternalCode`, without which no mapper runs:

```jsonc
{
  "compilerOptions": {},
  "contentMappers": [{ "package": "@unmango/tdl-typescript", "extensions": [".tdl"] }]
}
```

The package's `package.json` names the command `tsc` spawns, with the package's directory as its working directory:

```json
{ "typescript": { "contentMapper": { "exec": ["tdl-gen-typescript", "content-mapper"], "compilerOptions": [] } } }
```

The command speaks JSON-RPC 2.0 on standard input and output, framed as the Language Server Protocol frames it, with a `Content-Length` header.
Four requests reach it:

- `initialize`, answered with `protocolVersion` 1, `positionEncoding` `"utf-8"`, and `diagnosticSource`.
  Builds from 2026-09-18 on refuse a mapper that answers with another protocol version.
- `openProject`, carrying the tsconfig entry's `options`, answered with `optionDiagnostics`, each with a `path` into the options, `messageText`, and `code`.
- `transform`, carrying `fileName` and `content`, answered with the module: its `extension` (`.mts`), its `text`, and `diagnostics`, each with `messageText`, `code`, `start`, and `length` in the encoding `initialize` chose.
- `closeProject`.

The 7.1 compiler also reads `mappings`, `dependencies`, and a diagnostic `category` from a mapper, which Motherload does not send.
Their shapes are inferred from the compiler's messages: a mapping places generated text at an offset of the original content, and the dependencies name files whose change invalidates the transform.
This design uses all three, and falls back to what it says under each heading if one turns out to mean something else.

## The typescript backend owns it

The mapper is the `typescript` backend's, in `backend/typescript/mapper.go`, and `tdl-gen-typescript content-mapper` serves it.
With no argument the binary stays the plugin it is today.
`nix/cmd.nix` already ships the binary, so a Nix user has the command once `tdl` is installed.

A transform lowers the file in process and runs the backend over the request `tdl gen` would send it:

```text
tsc ──transform(fileName, content)──▶ tdl-gen-typescript content-mapper
                                         │ parser, sema: the file, its imports from disk
                                         │ analysis.Request, CheckDirectives
                                         │ Backend.Generate (mapper mode)
tsc ◀──────────── .mts text, diagnostics ┘
```

The binary already links the parser and `sema` through `internal/unlower`.
What it lacks is the host side: holding directives to their specs and silencing allowed loss codes.
Both live in `internal/gen`, which imports every backend for its registry, so the backend cannot import it.
They move to `internal/host`, below both, with no change in behavior; `internal/gen` calls them there.

A long-lived process is what the protocol gives, so the mapper keeps the lowered dependencies of the files it has transformed, keyed by path and modification time, for the life of a project.

## The module

One `.tdl` file is one module, holding the declarations that file declares, as the `typescript` backend writes them.
The module is types only: interfaces, unions, and aliases, every one exported, and no runtime code.
A TypeScript file imports it with `import type`; under `verbatimModuleSyntax`, a plain import of it is the compiler's own error.

A file needs no `typescript` target block.
When it has one, its directives apply as they do under `tdl gen` (`discriminant`, `foreign`, and the declaration directives), except `out` and `file`, which place files the mapper does not write.
A directive in another target's block is ignored, as `tdl gen --target typescript` ignores it.

A declaration from another TDL file is imported from that file:

```tdl
package shop

import "money.tdl" as money

value LineItem {
  price: money.Money
}
```

```ts
import type { Money } from "./money.tdl";

export interface LineItem {
  price: Money;
}
```

The specifier is the import's path relative to the importing file, so `tsc` resolves it to the dependency and asks the mapper for that module too.
This is a third way the backend's `extern.go` names an extern's module, beside a `foreign` directive and a dependency's `out` directive, and it is used only in mapper mode.
A `foreign` directive on the extern still wins.
A prelude name maps to a TypeScript type and imports nothing.

A transform names the files it read in `dependencies`, so `tsc` asks again when an imported `.tdl` file changes.
If the field means something else, an edit to `money.tdl` reaches `shop.tdl`'s diagnostics when `shop.tdl` is next transformed.

Either way, a dependency is read from disk.
The content `tsc` sends is the file being transformed, and an unsaved edit to another `.tdl` file reaches it only once saved.

## Diagnostics

Every diagnostic is reported under the source `tdl`, at the range `lex` positions give, which are already UTF-8 byte offsets of the content.

- A syntax or lowering error in the file is an error at its range, and the module is `export {}`, so an import of it names no type and only the `.tdl` error explains why.
- An error in a dependency is one error at the import that reaches it.
- A directive refused by `CheckDirectives` is an error at the directive.
- A declaration the backend refuses is an error at the declaration, and the cascade skips what names it, as under `tdl gen`.
- A loss warning is reported unless `tdl.toml`'s `[lossy]` table allows its code for `typescript`, as `tdl gen` drops it; the message ends with the code.
  It is a warning if the protocol's `category` carries one, and otherwise an error, so a project that wants a clean check allows the code.

The diagnostic code is a number, so each TDL diagnostic kind gets one, listed in the backend's page.

The mapper takes no options.
A key in the tsconfig entry's `options` is an option diagnostic naming it.

## Navigation

Go to definition from TypeScript lands where the module's mappings place it.
The backend records, as it writes each declaration, field, and variant, the range of the TDL source it came from, and the mapper sends them as `mappings`, so a jump from `Order` lands on `entity Order` in `shop.tdl`.
Without mappings, the jump lands in the generated text, which an editor shows as a read-only document; navigation is then a backlog item, not a blocker.

## Runtime

A type-only import is erased, so nothing loads a `.tdl` file at run time, and no Node, Bun, or bundler loader is needed.
Runtime values, schemas a program validates JSON against, belong to the [zod backend](zod-backend.md); a mapper option serving them is in the [backlog](../backlog.md).

## The npm package

`tsc` finds a mapper by package name, so the command needs a package to name it.
`@unmango/tdl-typescript` is a `package.json` and a README, with no JavaScript, in `npm/tdl-typescript/`.
It does not carry the binary: `tdl-gen-typescript` comes from `PATH`, as the C# source generator takes `tdl` from it.
Platform packages carrying the binary, as esbuild publishes them, are in the backlog.

`release-please-config.json` versions it with the release, and a second job in `release-please.yml` publishes it with npm trusted publishing, gated on the release job's output as the NuGet job is.
The npm trusted publisher names `release-please.yml`; the scope's owner configures it before the first release.

## Testing

- A Go test in `backend/typescript` drives the mapper over a pipe: initialize, open a project, transform the smoke model's files and a file importing another, and close.
  The text of each module equals what `tdl gen --target typescript` writes, but for the imports, and each diagnostic kind is asserted at its range.
- A second test runs `tsc --noEmit --runExternalCode` over `backend/typescript/testdata/mapper/`, a TypeScript project importing `.tdl` files, when a `tsc` of 7.1 or later is on `PATH`, and skips otherwise.
  It points the project at `tdl-gen-typescript` built by the test.
- `checks.typescript-content-mapper` runs the same project once nixpkgs carries TypeScript 7.1.

## Not designed here

- Runtime values through the zod backend.
- Platform npm packages carrying `tdl-gen-typescript`.
- Unsaved dependency buffers: the protocol sends one file's content.
- A mapper for any other backend; the protocol is TypeScript's.
