# Foreign imports

Design document.
Nothing here is built; [foreign-imports-plan.md](foreign-imports-plan.md) orders the work.

A foreign import names a file in a target language instead of a `.tdl` file:

```tdl
package shop

import "types.go" as gotypes
import "api.ts" as api

value Order {
  id:       gotypes.OrderID
  customer: api.Customer
}
```

The file is read by its target's reverse backend each time the model is lowered, and the result is lowered in memory like any other dependency.
Nothing is written to disk.
`tdl import` stays for a project that wants to own the TDL; a foreign import is for a project whose source of truth is the other language.

## Choosing the reader

The reader is chosen by the file's extension.

| Extension | Target |
| --- | --- |
| `.go` | `go` |
| `.proto` | `protobuf` |
| `.thrift` | `thrift` |
| `.graphql`, `.graphqls` | `graphql` |
| `.smithy` | `smithy` |
| `.ts` | `typescript` |

A backend claims extensions in its handshake, in `Features.extensions` beside `Features.reverse`, so a `tdl-gen-<name>` on `PATH` takes part exactly as a built-in does.
A backend declaring extensions without `reverse` is refused at the handshake.

`from` names the target explicitly and overrides the extension:

```tdl
import "openapi.json" from openapi as api
```

`from` is required when more than one backend claims the extension, and the error lists every claimant.
An extension no backend claims, without `from`, is an error naming the extension.
`from` is contextual: it is special only between an import's path and `as`, so `from: string` stays a field.
`.tdl` is never a foreign extension, and `from` on a `.tdl` path is an error.

## What the import binds

A path names one file, for every target.
The Go reader reads the rest of the file's package only to resolve names the file uses; it binds the file's declarations alone.

The imported package is the source's own: the Go `package` clause, the `.proto` package, the `.ts` file's stem, the rule `tdl import` follows without `--package`.
The alias is only the local name it binds to, as for a `.tdl` import, and `_` merges the declarations into the importing scope.

Only declarations the source exports can be named, judged by TDL's capitalization rule on the names the reader gives them.
A private source declaration that an exported one needs is still lowered, and stays unnameable.

A name the foreign file takes from another file it imports, `.ts` importing `other.ts` or `.proto` importing another `.proto`, is not followed.
It is a diagnostic on the `import` line naming the unresolved name; importing the other file too is the remedy.
Following foreign imports transitively is in [backlog.md](../backlog.md#foreign-imports-that-import).

## Lowering

`sema.Loader` gains a second result for a foreign path: the target and the bytes, instead of TDL source.
`sema` sends an `ImportRequest` through the backend registry, takes the model or TDL source the response carries, unlowers a model with `internal/unlower`, and lowers the result as the imported package.
Lowering stays free of the filesystem and of processes: the loader and the registry supply both, and tests use `sema.MapLoader` with an in-process registry.

Reader diagnostics are reported at the `import` line, with the reader's position in the foreign file attached as related information.
A warning carrying a loss code is dropped when `tdl.toml` or `--allow-lossy` allows it, under `[lossy]` or `[lossy.<target>]` for the reader's target, exactly as `tdl import` drops it.
An error from the reader fails lowering.

A reader that needs a tool, `node` and `tsc` for TypeScript or the Smithy CLI, reports a missing one as an error at the `import` line naming the tool.
A foreign import is never skipped silently, because every declaration behind it would vanish with it.

## Generating the source's own target

A model importing `types.go` and generating Go refers to `gotypes.OrderID` as the original Go type; nothing is regenerated.

The import response gains `source`: target block directives that place the source file for its own target.
The Go reader returns `package` with the import path, read from the nearest `go.mod` above the file and the file's directory.
The TypeScript reader returns `out` with the file's path.
Each other reader returns what its forward twin reads to refer to a dependency.

`sema` lowers `source` as the imported package's block for the reader's target, so the extern machinery of [ir-plan.md](ir-plan.md) phase 8b points at the original with nothing new in any backend.
A reader that cannot place its file, such as a Go file with no `go.mod` above it, warns, and the source target then vendors the declarations like any other.

A target block directive on a foreign declaration for its own source target is an error, since nothing is generated for it to apply to.

## Generating other targets

For every other target the declarations are vendored: generated as their own unit beside the importer's output, never merged into the importer's package, where they could collide with its names and lose their qualifier.

The unit is placed under the importer's `out`, at a path named for the source's package.

| Target | Vendored unit for a source in package `types`, imported by package `example.com/shop` |
| --- | --- |
| `go` | package `example.com/shop/types`, in `types/` |
| `typescript` | `types.ts` beside the importer's file |
| `protobuf` | `types.proto` in the importer's directory, package `types` |
| others | the file their forward backend writes for a package of that name |

Vendoring warns once per foreign import and target, naming the source file, with the new loss code `lossy.vendored`, so a project that accepts it silences it under `[lossy]`.
`lossy.vendored` joins the table in [reverse.md](reverse.md) and `emit.LossCodes`.

A target block may put directives on a vendored declaration with the path it is referred to by, as on a `_`-merged dependency's declaration today:

```tdl
typescript for shop {
  gotypes.Order => name("ShopOrder")
}
```

The directive lands on the declaration's `ir.Extern` and on the vendored declaration itself.

## Language server

`internal/lsp` caches each foreign file's lowered package by the content hash of the file, keyed also by the reader's target, and runs the reader again only when the hash changes.
The cache is in memory only, so nothing is written beside the source and a restart starts clean.
An open foreign file's unsaved text is what the overlay loader serves, so the hash follows the editor.

Hover and go to definition on `gotypes.Order` jump into `types.go` at the position the reader reports.
A reader that reports no position for a declaration jumps to the top of the file.

## Watching

`tdl gen --watch` and `tdl play` watch every foreign file a model imports, as they watch `.tdl` imports.

## Nix

The flake module's `tdl-check` and `tdl-gen` checks gain a `tools` option: packages put on `PATH` while they run, empty by default.
A project importing `.ts` adds `nodejs` and `typescript`; one importing `.smithy` adds `smithy-cli`.
The option is not inferred from extensions, so every input of a check is written in its consumer's flake.

## Not in this design

- Following one foreign file's imports into another, which is backlog.
- A directory as a path, even for Go.
- Writing a cache to disk.
- Readers for targets that have none yet: jsonschema, openapi, salesforce, haskell, csharp, and likec4 gain foreign imports when they gain `reverse`, with no further design.
