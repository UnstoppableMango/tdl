# tdl

[![CI](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml/badge.svg)](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml)
[![Codecov](https://img.shields.io/codecov/c/github/UnstoppableMango/tdl)](https://app.codecov.io/gh/UnstoppableMango/tdl)
[![Built with Nix](https://img.shields.io/badge/Built%20with-Nix-5277C3?logo=nixos&logoColor=white)](https://nixos.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/unstoppablemango/tdl.svg)](https://pkg.go.dev/github.com/unstoppablemango/tdl)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Last commit](https://img.shields.io/github/last-commit/UnstoppableMango/tdl)](https://github.com/UnstoppableMango/tdl/commits/main)
[![Hercules CI](https://hercules-ci.com/api/v1/site/github/account/UnstoppableMango/project/tdl/badge)](https://hercules-ci.com/github/UnstoppableMango/tdl)

TDL is a language for describing domain models: what things are, what identifies them, how they relate, and what values they may hold.
It compiles a model into equivalent definitions in other formats, such as Go, protobuf, or GraphQL.
It has no expressions, control flow, or runtime.

![tdl generating Go and TypeScript from a model](docs/demo/demo.gif)

Made with [VHS](https://github.com/charmbracelet/vhs) ([view source](docs/demo/demo.tape)).

This repository holds the [language specification](docs/spec.md) and its reference implementation in Go.

## Status

Early, incomplete, and changing.

- **Front end: done.** The lexer and parser read the whole [grammar](docs/grammar.ebnf).
- **Resolved model: nearly done.** `tdl ir` resolves names, imports, mixins, class satisfaction, constraints, defaults, units, and target directives. Merging a dependency's target blocks is partial.
- **Code generation:** Go, GraphQL, JSON Schema, OpenAPI, protobuf, Salesforce, Smithy, Thrift, TypeScript, and Zod. `tdl gen` also runs any `tdl-gen-<name>` plugin on `PATH`.
- **Editors:** a language server, a tree-sitter grammar for Neovim, and a VS Code extension.

## Documents

| Document | Covers |
| --- | --- |
| [spec.md](docs/spec.md) | The language. Canonical. |
| [grammar.ebnf](docs/grammar.ebnf) | The formal grammar. |
| [architecture/identity.md](docs/architecture/identity.md) | Identity and the `Entity` class. |
| [architecture/ir.md](docs/architecture/ir.md) | The resolved model backends consume. |
| [architecture/plugins.md](docs/architecture/plugins.md) | The backend plugin protocol. |
| [architecture/go-backend.md](docs/architecture/go-backend.md) | The Go backend. |
| [architecture/schema-backends.md](docs/architecture/schema-backends.md) | The protobuf, Thrift, Smithy, GraphQL, TypeScript, JSON Schema, and OpenAPI backends. |
| [architecture/salesforce-backend.md](docs/architecture/salesforce-backend.md) | The Salesforce backend. |
| [architecture/lsp.md](docs/architecture/lsp.md) | The language server. |
| [architecture/treesitter.md](docs/architecture/treesitter.md) | Deriving the tree-sitter grammar from the EBNF. |
| [architecture/editors.md](docs/architecture/editors.md) | Highlighting in Neovim, VS Code, Zed, and on GitHub. |
| [backends/](docs/backends/README.md) | What each backend generates, feature by feature. |
| [Issues](https://github.com/UnstoppableMango/tdl/issues) | Planned and wanted work: a spec issue per feature, with its phases as sub-issues. |

`docs/architecture/` describes what is built; `docs/adr/` records decisions from ADR-0001 on.

## Example

```tdl
package shop

type Order: Entity {
  id: OrderId
  customer: Customer
  shipping: Address?
  items: [LineItem] owned where { length(1..) }
  status: Status = Draft
  total: Money
}

type Customer: Entity {
  email: Email
  name: string?
}

type Address {
  line1: string
  line2: string?
  city: string
  postcode: string
  country: string
}

type Money {
  amount: decimal
  currency: Currency
}

type OrderId: uuid

type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
}

enum Currency { USD EUR GBP }

enum Status { Draft Placed Shipped Cancelled }
```

`Order` conforms to `Entity`, so it has identity that survives its contents changing; `Address` does not.
What a code generator needs goes in a separate `target` block, never in the model.

## Install

Run it without installing:

```shell
nix run github:UnstoppableMango/tdl
```

Or install it with `nix profile install github:UnstoppableMango/tdl`.

### NixOS or home-manager

`overlays.default` adds `pkgs.tdl`, `pkgs.tdl-gmk`, and `pkgs.vscode-tdl`, and includes the [gomod2nix](https://github.com/nix-community/gomod2nix) overlay it builds with.
It builds `tdl` with `go_1_27`, so the nixpkgs it is applied to has to provide that attribute.
On an older nixpkgs, use `inputs.tdl.packages.${system}.default`, which builds against the nixpkgs tdl's lock file pins, and do not set `inputs.tdl.inputs.nixpkgs.follows`.

```nix
{
  inputs.tdl.url = "github:UnstoppableMango/tdl";

  # ... in a NixOS or home-manager configuration:
  nixpkgs.overlays = [ inputs.tdl.overlays.default ];
  environment.systemPackages = [ pkgs.tdl ];
}
```

The home-manager module (`homeModules.default`, also `homeManagerModules.default`) adds `programs.tdl.enable` for the CLI and `programs.tdl.vscode.enable` for the VS Code extension.

```nix
{
  imports = [ inputs.tdl.homeModules.default ];

  nixpkgs.overlays = [ inputs.tdl.overlays.default ];
  programs.tdl.enable = true;
}
```

### In a project

`flakeModules.default` is a [flake-parts](https://flake.parts) module for a project that contains `.tdl` files.
It adds `devShells.tdl` (pull it into your shell with `inputsFrom`) and checks that each model parses and lowers, is canonically formatted, and, for `gen.files`, that generated output on disk is current.
`files` are strings relative to `src`, so `include` paths keep resolving.

```nix
{
  imports = [ inputs.tdl.flakeModules.default ];

  perSystem = { system, ... }: {
    _module.args.pkgs = import inputs.nixpkgs {
      inherit system;
      overlays = [ inputs.tdl.overlays.default ];
    };

    tdl = {
      enable = true;
      src = ./model;
      files = [ "orders.tdl" "billing.tdl" ];
      gen.files = [ "orders.tdl" ];
    };
  };
}
```

Set `tdl.fmt.enable = false` to skip the formatting check.

### In a Makefile

`tdl-gmk` is a [GNU make loadable object](https://www.gnu.org/software/make/manual/html_node/Loading-Objects.html), `lib/tdl.so`, that defines two functions:

- `$(tdl-files dir...)`: the `.tdl` files under each dir.
- `$(tdl-imports file...)`: each file and every file it imports, transitively.

Both return sorted paths relative to the directory of the Makefile that loaded the object, wherever make runs from.
A failure, such as an import that does not exist, stops make with the error.

`devShells.tdl` sets `TDL_GMK` to the object and puts nixpkgs' `gnumake` on `PATH`, since the object links against nixpkgs' libc and a system make cannot load it.
Set `tdl.make.enable = false` to leave both out.

```make
load $(TDL_GMK)

MODEL := $(tdl-files model)

proto/.stamp: $(tdl-imports $(MODEL))
	tdl gen $(MODEL)
	touch $@
```

`tdl gen --make $@` writes the stamp itself, as a make dependency file in the style of `gcc -MD -MP`: a rule naming the target and every file the run read, imports and `tdl.toml` included, then an empty rule for each, so deleting an import rebuilds rather than fails.
It is written only after every file has generated, or verified, with no error, so a failed run leaves it stale and make retries.
A Makefile that includes it needs no list of imports:

```make
load $(TDL_GMK)

MODEL := $(tdl-files model)

proto/.stamp: $(MODEL)
	tdl gen $(MODEL) --make $@

-include proto/.stamp
```

Without Nix, build it with `go build -tags makeplugin -buildmode=c-shared -o tdl.so ./cmd/tdl-gmk` from a checkout, with `gnumake.h` on the include path.
make derives the setup function from the file's name, so keep it `tdl.so`.

## Usage

```shell
tdl check ./types.tdl    # parse and lower, and report every problem
tdl fmt ./types.tdl      # print canonical formatting; -w writes in place
                         # --check lists what is not canonical and exits non-zero
tdl ast ./types.tdl      # print the parse tree
tdl gen ./types.tdl      # run every target block; --target narrows, -o overrides
                         # --verify checks, --clean removes old output, --watch reruns
                         # --make writes a make dependency file for a target
tdl ir ./types.tdl       # print the resolved model; --format json for the plugin view
                         # --prelude lowers against a replacement prelude
tdl tokens ./types.tdl   # print the token stream
tdl version              # tool and spec versions
```

Commands accept several files and report every failing file, not only the first.
With more than one file, output is separated by `==> path <==` banners.
A file named `-` is standard input, so `tdl fmt -` formats an unsaved editor buffer.
`fmt -w` and `gen` reject `-`, since there is nothing to write back to and no directory to resolve imports from.

### Playground

`tdl play` watches a file and re-renders it on every save.

```shell
tdl play                              # scratch.tdl, created from a template if missing
tdl play ./types.tdl --views all      # source, fmt, ast, tokens, stats
tdl play ./types.tdl --once           # render and exit
```

Views are `source`, `fmt`, `ast`, `tokens`, `stats`, or `all`; the default is `fmt,ast`.
Parse errors show below the panes with a caret at the column.

[`examples/`](examples/README.md) has files to start from.

## Editor support

Highlighting comes from two grammars generated from [grammar.ebnf](docs/grammar.ebnf); [architecture/editors.md](docs/architecture/editors.md) explains why there are two.

### Neovim

The parser and queries are in `tree-sitter/`.
On `nvim-treesitter`'s `main` branch, register the parser and start highlighting:

```lua
vim.filetype.add({ extension = { tdl = 'tdl' } })

vim.api.nvim_create_autocmd('User', {
  pattern = 'TSUpdate',
  callback = function()
    require('nvim-treesitter.parsers').tdl = {
      install_info = {
        url = 'https://github.com/UnstoppableMango/tdl',
        location = 'tree-sitter',
        queries = 'tree-sitter/queries',
      },
    }
  end,
})

vim.api.nvim_create_autocmd('FileType', {
  pattern = 'tdl',
  callback = function() vim.treesitter.start() end,
})
```

Then run `:TSInstall tdl` (needs the `tree-sitter` CLI on `PATH`), and `:TSUpdate tdl` to pick up later revisions.

On the `master` branch, enable highlighting with `highlight = { enable = true }` in `setup`, and register the parser through `require('nvim-treesitter.parsers').get_parser_configs().tdl` with `files = { 'src/parser.c', 'src/scanner.c' }` in place of `queries`.
That branch installs no queries for a custom parser, so copy `tree-sitter/queries/highlights.scm` to `queries/tdl/highlights.scm` on your runtimepath.

### VS Code

`nix build .#vscode-tdl` builds the extension in [editors/vscode](editors/vscode).
Install it with `programs.tdl.vscode.enable`, by adding `pkgs.vscode-tdl` to `vscode-with-extensions` or `programs.vscode.profiles.<name>.extensions`, or with `make vscode-install`.

The extension runs `tdl lsp` for diagnostics, hover, go to definition, formatting, and the outline.
The nix build points it at its own `tdl`; otherwise it runs `tdl` from `PATH`, or the `tdl.server.path` setting.
Without a server, highlighting still works.

## Support matrix

`Front end` is the lexer, parser, and `tdl fmt`; `IR` is `tdl ir`, the resolved model a backend consumes.

| Construct | Front end | IR |
| --- | --- | --- |
| `package`, `import` | Yes | Yes, resolved across packages without inlining |
| `primitive` | Yes | Yes |
| `alias` | Yes | Yes |
| `type` (newtype chains) | Yes | Yes, constraints accumulate down the chain |
| `type` with a body, `: Entity` | Yes | Yes, the kind computed from conformance |
| `mixin`, `include` | Yes | Yes, expanded |
| `enum`, variants with fields | Yes | Yes |
| `class`, functional dependencies, associated types | Yes | Yes |
| `instance`, including conditional instances | Yes | Yes |
| Type parameters and kinds | Yes | Yes, parameters stay parameters |
| Collection and option sugar (`[T]`, `{T}`, `{K -> V}`, `T?`, `T \| null`) | Yes | Yes, lowered to whatever the prelude declares |
| `where` constraints | Yes | Yes, open set: standard names checked, others passed through |
| Field defaults | Yes | Yes, resolved against the field's type |
| `deprecated` | Yes | Yes |
| `unit` | Yes | Yes, reduced to base dimensions |
| `target` blocks | Yes | Partial: a dependency's declaration-level directives are not merged |

`tdl fmt` keeps both comment forms: a `///` doc comment attaches to the next declaration, and a `//` comment stays on its own line or at the end of its line.
Inside a body, the formatter decides blank lines.

### Backends

Each built-in backend also ships as a `tdl-gen-<name>` plugin.
[docs/backends/](docs/backends/README.md) has a page per backend saying what it does with each feature of the language.

| Backend | Generates |
| --- | --- |
| [`go`](docs/backends/go.md) | Structs, entity keys, both enum shapes, newtypes, generics, classes as interfaces, `Validate` methods, foreign types |
| [`graphql`](docs/backends/graphql.md) | Output types, both enum shapes, custom scalars, lists; no maps |
| [`jsonschema`](docs/backends/jsonschema.md) | A JSON Schema document: definitions, both enum shapes, constraints as keywords |
| [`openapi`](docs/backends/openapi.md) | An OpenAPI 3.1, 3.0, or 2.0 document of schemas in YAML or JSON: both enum shapes (no fielded enums in 2.0), constraints as keywords; no paths |
| [`protobuf`](docs/backends/protobuf.md) | Messages, both enum shapes, newtypes, collections, `number` pins, services |
| [`salesforce`](docs/backends/salesforce.md) | Salesforce DX source: a custom object per entity, Apex for values and enums |
| [`smithy`](docs/backends/smithy.md) | Structures, both enum shapes, named collection shapes |
| [`thrift`](docs/backends/thrift.md) | Structs, both enum shapes, newtypes as typedefs, collections, `number` pins |
| [`typescript`](docs/backends/typescript.md) | JSON wire types: interfaces, both enum shapes, newtypes as aliases |
| `debug` | A description of the model it was given |
| Anything else | `tdl-gen-<name>` on `PATH`, over the [plugin protocol](docs/architecture/plugins.md) |

What each backend generates from each feature of the language, measured by `make features`.
Yes generates it, Warns loses part of it under a loss code that `tdl.toml` can allow, No skips it or refuses the model, and Planned names a phase that will generate it.
A backend's page names the codes and phases.
Warns counts only what the output loses: what a later `tdl import` would not read back is reported with a bare `readback` directive in the target block.

<!-- tdl:features generate -->

| Feature | `csharp` | `go` | `graphql` | `haskell` | `jsonschema` | `likec4` | `openapi` | `protobuf` | `salesforce` | `smithy` | `thrift` | `typescript` | `zod` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Entity, value, and mixin | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Mixin `include` | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Fieldless enum | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Fielded enum | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Newtype | Yes | Yes | Warns | Yes | Yes | Yes | Yes | Warns | Yes | Yes | Yes | Yes | Yes |
| Alias | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Optional and nullable | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| List, set, and map | Yes | Yes | No | Yes | Yes | Yes | Yes | Warns | Yes | Yes | Yes | Warns | Yes |
| `min` | Planned | Yes | Warns | Planned | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Warns | Planned |
| `max` | Planned | Yes | Warns | Planned | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Warns | Planned |
| `length` | Planned | Yes | Warns | Planned | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Warns | Planned |
| `matches` | Planned | Yes | Warns | Warns | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Warns | Planned |
| `oneOf` | Planned | Yes | Warns | Planned | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Yes | Planned |
| `unique` | Planned | Yes | Warns | Planned | Yes | Yes | Yes | Warns | Warns | Warns | Warns | Warns | Planned |
| `owned` | Yes | Warns | Warns | Yes | Yes | Yes | Yes | Warns | No | Warns | Warns | Warns | Yes |
| Field default | Yes | Warns | Warns | Yes | Yes | Yes | Yes | Warns | Yes | Warns | Warns | Warns | Yes |
| `key` directive | Yes | Yes | No | Yes | No | No | No | No | Yes | No | No | No | No |
| `name` directive | Yes | Yes | Yes | Yes | Yes | No | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `number` directive | No | No | No | No | No | No | No | Yes | No | No | Yes | No | No |
| Doc comment | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Type parameters | Planned | Yes | No | Yes | No | Yes | No | No | No | No | No | No | Planned |
| Class | Planned | Warns | Warns | Yes | No | Yes | No | Warns | No | Warns | Warns | Warns | Planned |
| Unit | Warns | Planned | Warns | Planned | No | Yes | No | Warns | No | Warns | Warns | Warns | Planned |
| Import and extern | Yes | Yes | No | Yes | No | Yes | No | Yes | No | No | No | Yes | Yes |

<!-- /tdl:features -->

## Development

```shell
command make build   # nix build .#
command make test    # go test ./...
command make lint    # nix flake check + golangci-lint + buf + markdownlint
command make fmt     # nix fmt + buf format
```

Without Nix, `go build ./...` and `go test ./...` work directly.
[AGENTS.md](AGENTS.md) has the full command list and architecture.

Releases come from [release-please](https://github.com/googleapis/release-please); never edit a version or `CHANGELOG.md` by hand.

## Design principles

- The core is small. Most of what looks like a type system is TDL code in a replaceable prelude.
- Identity is first class, and the model is pure: what a backend needs lives in a `target` block.
- Constraints are syntax. The compiler parses and resolves them; backends decide what they mean.
- The grammar is small and strict, with a hand-written lexer and parser.
- The spec and the plain-text `testdata/conformance` and `testdata/invalid` corpora are the contract another implementation would satisfy.
