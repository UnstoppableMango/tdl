# tdl

[![CI](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml/badge.svg)](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml)
[![Codecov](https://img.shields.io/codecov/c/github/UnstoppableMango/tdl)](https://app.codecov.io/gh/UnstoppableMango/tdl)
[![Built with Nix](https://img.shields.io/badge/Built%20with-Nix-5277C3?logo=nixos&logoColor=white)](https://nixos.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/unstoppablemango/tdl.svg)](https://pkg.go.dev/github.com/unstoppablemango/tdl)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Last commit](https://img.shields.io/github/last-commit/UnstoppableMango/tdl)](https://github.com/UnstoppableMango/tdl/commits/main)
[![Hercules CI](https://hercules-ci.com/api/v1/site/github/account/UnstoppableMango/project/tdl/badge)](https://hercules-ci.com/github/UnstoppableMango/tdl)

TDL is a language for describing domain models: what things are, what identifies them, how they relate, and what values they may hold.
It compiles a model into equivalent definitions in other formats, such as Go, protobuf, or GraphQL.
It has no expressions, control flow, or runtime.

This repository holds the [language specification](docs/spec.md) and its reference implementation in Go.

## Status

Early, incomplete, and changing.

- **Front end: done.** The lexer and parser read the whole [grammar](docs/grammar.ebnf).
- **Resolved model: nearly done.** `tdl ir` resolves names, imports, mixins, class satisfaction, constraints, defaults, units, and target directives. Merging a dependency's target blocks is partial.
- **Code generation:** Go, GraphQL, protobuf, Salesforce, Smithy, Thrift, and TypeScript. `tdl gen` also runs any `tdl-gen-<name>` plugin on `PATH`.
- **Editors:** a language server, a tree-sitter grammar for Neovim, and a VS Code extension.

## Documents

| Document | Covers |
| --- | --- |
| [spec.md](docs/spec.md) | The language. Canonical. |
| [grammar.ebnf](docs/grammar.ebnf) | The formal grammar. |
| [design/workflow.md](docs/design/workflow.md) | What a model author does with all of it. |
| [design/identity.md](docs/design/identity.md) | Identity and the `Entity` class. |
| [design/ir.md](docs/design/ir.md) | The resolved model backends consume. |
| [design/plugins.md](docs/design/plugins.md) | The backend plugin protocol. |
| [design/go-backend.md](docs/design/go-backend.md) | The Go backend. |
| [design/schema-backends.md](docs/design/schema-backends.md) | The protobuf, Thrift, Smithy, GraphQL, and TypeScript backends. |
| [design/salesforce-backend.md](docs/design/salesforce-backend.md) | The Salesforce backend. |
| [design/lsp.md](docs/design/lsp.md) | The language server. |
| [design/treesitter.md](docs/design/treesitter.md) | Deriving the tree-sitter grammar from the EBNF. |
| [design/editors.md](docs/design/editors.md) | Highlighting in Neovim, VS Code, Zed, and on GitHub. |
| [backlog.md](docs/backlog.md) | Wanted, unscheduled work. |

A `*-plan.md` beside a design tracks its implementation.

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

`overlays.default` adds `pkgs.tdl` and `pkgs.vscode-tdl`, and includes the [gomod2nix](https://github.com/nix-community/gomod2nix) overlay it builds with.

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
It adds `devShells.tdl` (pull it into your shell with `inputsFrom`) and checks that each model parses, is canonically formatted, and, for `gen.files`, that generated output on disk is current.
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

## Usage

```shell
tdl check ./types.tdl    # parse and report syntax errors
tdl fmt ./types.tdl      # print canonical formatting; -w writes in place
                         # --check lists what is not canonical and exits non-zero
tdl ast ./types.tdl      # print the parse tree
tdl gen ./types.tdl      # run every target block; --target narrows, -o overrides
                         # --verify checks, --clean empties first, --watch reruns
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

Highlighting comes from two grammars generated from [grammar.ebnf](docs/grammar.ebnf); [design/editors.md](docs/design/editors.md) explains why there are two.

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

| Backend | Generates |
| --- | --- |
| `go` | Structs, entity keys, both enum shapes, newtypes, generics, classes as interfaces, `Validate` methods, foreign types |
| `graphql` | Output types, both enum shapes, custom scalars, lists; no maps |
| `protobuf` | Messages, both enum shapes, newtypes, collections, `number` pins, services |
| `salesforce` | Salesforce DX source: a custom object per entity, Apex for values and enums |
| `smithy` | Structures, both enum shapes, named collection shapes |
| `thrift` | Structs, both enum shapes, newtypes as typedefs, collections, `number` pins |
| `typescript` | JSON wire types: interfaces, both enum shapes, newtypes as aliases |
| `debug` | A description of the model it was given |
| Anything else | `tdl-gen-<name>` on `PATH`, over the [plugin protocol](docs/design/plugins.md) |

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
