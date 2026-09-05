# tdl

[![CI](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml/badge.svg)](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml)
[![Codecov](https://img.shields.io/codecov/c/github/UnstoppableMango/tdl)](https://app.codecov.io/gh/UnstoppableMango/tdl)
[![Built with Nix](https://img.shields.io/badge/Built%20with-Nix-5277C3?logo=nixos&logoColor=white)](https://nixos.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/unstoppablemango/tdl.svg)](https://pkg.go.dev/github.com/unstoppablemango/tdl)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Last commit](https://img.shields.io/github/last-commit/UnstoppableMango/tdl)](https://github.com/UnstoppableMango/tdl/commits/main)

TDL is a language for describing domain models.
It says what things are, what identifies them, how they relate, and what values they may hold, and compiles them into equivalent definitions in other structured formats.
It describes no behavior and has no expressions, control flow, or runtime.

This repository owns the canonical [language specification](docs/spec.md) and its reference implementation, written in Go.

## Status

Early and incomplete, and moving.

**The front end is done.** The lexer and parser read the whole grammar, and `tdl check`, `tdl fmt`, `tdl ast`, and `tdl tokens` work across it.
Nothing in [grammar.ebnf](docs/grammar.ebnf) is missing.

**The middle is most of the way there.** `tdl ir` prints a resolved model: names, imports, mixins, class satisfaction, constraints, defaults, and target directives, all resolved.
The support matrix below says what each construct reaches.
Units, and merging a dependency's target blocks, are the two pieces still outstanding.

**The back end runs, with nothing behind it yet.** `tdl gen` resolves a target's backend to a built-in or to `tdl-gen-<name>` on PATH, sends it the resolved model, and writes back the files it returns.
The only backend is `debug`, which prints what it was given, so nothing generates real code yet.

The design is settled and written down:

| Document | What it covers |
| --- | --- |
| [spec.md](docs/spec.md) | The language. Canonical. |
| [grammar.ebnf](docs/grammar.ebnf) | The formal grammar. |
| [design/parser-plan.md](docs/design/parser-plan.md) | Rewriting the lexer and parser to match. Done. |
| [design/ir.md](docs/design/ir.md) | The resolved model backends consume. |
| [design/ir-plan.md](docs/design/ir-plan.md) | Implementing it. Everything but dependency target merging. Units are out of its scope. |
| [design/plugins.md](docs/design/plugins.md) | The backend plugin protocol. |
| [design/plugins-plan.md](docs/design/plugins-plan.md) | Implementing it. Done, but for replaying the recorded exchanges against a non-Go implementation. |
| [design/treesitter.md](docs/design/treesitter.md) | Deriving the tree-sitter grammar from the EBNF. |
| [design/treesitter-plan.md](docs/design/treesitter-plan.md) | Implementing it. Phases 1 to 3 of 7 done. |
| [design/editors.md](docs/design/editors.md) | Highlighting in Neovim, VS Code, Zed, and on GitHub. |
| [design/editors-plan.md](docs/design/editors-plan.md) | Implementing it. Phases 1 to 3 of 5 done. |
| [design/workflow.md](docs/design/workflow.md) | What a model author does with all of it. |
| [backlog.md](docs/backlog.md) | Wanted, unscheduled: `tdl fmt` as a `treefmt` formatter, an LSP, editor support, an MCP server. |

## Example

```tdl
package shop

entity Order {
  key id: OrderId
  customer: Customer
  shipping: Address?
  items: [LineItem] owned where { length(1..) }
  status: Status = Draft
  total: Money
}

entity Customer {
  key email: Email
  name: string?
}

value Address {
  line1: string
  line2: string?
  city: string
  postcode: string
  country: string
}

value Money {
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

`entity` and `value` is the modelling decision: an `Order` has identity that survives its contents changing, an `Address` does not.
Everything a code generator needs lives in a separate `target` block, never in the model.

## Install

`nix run github:UnstoppableMango/tdl` runs the CLI without installing it, and `nix profile install github:UnstoppableMango/tdl` installs it.

`overlays.default` is the way into a configuration.
It adds `tdl` and `vscode-tdl` to a nixpkgs instance, and composes [gomod2nix](https://github.com/nix-community/gomod2nix)'s overlay, which `tdl` is built with, so it is the only one to add.

```nix
{
  inputs.tdl.url = "github:UnstoppableMango/tdl";

  # ... in a NixOS or home-manager configuration:
  nixpkgs.overlays = [ inputs.tdl.overlays.default ];
  environment.systemPackages = [ pkgs.tdl ];
}
```

`homeModules.default` is a home-manager module over that overlay, exported as `homeManagerModules.default` under the older name as well.
`programs.tdl.enable` installs the CLI, and `programs.tdl.vscode.enable` adds the extension to the VS Code profiles named in `programs.tdl.vscode.profiles`, defaulting to `default` and to whether `programs.vscode` is enabled at all.

```nix
{
  imports = [ inputs.tdl.homeModules.default ];

  nixpkgs.overlays = [ inputs.tdl.overlays.default ];
  programs.tdl.enable = true;
}
```

`flakeModules.default` is for the other side: a [flake-parts](https://flake.parts) module a project whose sources include `.tdl` files imports into its own flake.
`tdl.enable` defines `devShells.tdl`, a shell to pull into an existing one with `inputsFrom`.
`tdl.files` names the models, relative to `tdl.src` rather than as paths, because an `include` resolves relative to the file that writes it.
`checks.tdl-check` parses each one and `checks.tdl-fmt` asserts it is canonically formatted; `tdl.gen.files` adds `checks.tdl-gen`, which runs `tdl gen --verify` and fails when generated output on disk is stale.

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

`tdl fmt` drops ordinary `//` comments, so a project that writes them sets `tdl.fmt.enable = false`.

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

Every command above takes more than one file, and reads standard input when handed `-`.
A file that fails is reported and the rest still run, so `tdl check ./*.tdl` names every broken file rather than the first.
The commands that print something separate their output with a `==> path <==` banner when given more than one, and print no banner for a single file.
`tdl fmt -` formats an editor buffer that has not been saved.
`-w` rejects it, having nothing to write back to, and so does `gen`: an import resolves next to the file that wrote it, so reading a model from standard input would resolve every import against the working directory instead.

### Playground

`tdl play` watches a file and re-renders it on every save.

```shell
tdl play                              # scratch.tdl, created from a template if missing
tdl play ./types.tdl --views all      # source, fmt, ast, tokens, stats
tdl play ./types.tdl --once           # render and exit
```

Views are `source`, `fmt`, `ast`, `tokens`, `stats`, or `all`; the default is `fmt,ast`.
Parse errors render below the panes with a caret at the reported column.

[`examples/`](examples/README.md) holds files to start from: the same domain modelled flat and nested, plus collections and target blocks.

## Editor support

Highlighting comes from two derived grammars, both generated from [grammar.ebnf](docs/grammar.ebnf).
[design/editors.md](docs/design/editors.md) says why there are two.

### Neovim

The parser and its queries live in `tree-sitter/`, which is the layout `nvim-treesitter` reads with `location` and `queries`.
The parser is built from this repository, since nothing is published yet.

On `nvim-treesitter`'s `main` branch, the parser is registered in a `User TSUpdate` autocommand:

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

The last block is what colors a buffer.
`nvim-treesitter` installs the parser and the queries and enables nothing, so highlighting is Neovim's `vim.treesitter.start`, called per filetype from an autocommand or from `ftplugin/tdl.lua`.

The parser is named for the filetype, so `vim.treesitter.language.register` is not needed.

Then `:TSInstall tdl`, which clones the repository, compiles the committed parser, and installs `highlights.scm` beside it.
That branch builds through the `tree-sitter` CLI, so it has to be on the PATH.
`:TSUpdate tdl` picks up a later revision.

On the `master` branch the highlighting is the plugin's rather than Neovim's, enabled with `highlight = { enable = true }` in its `setup`, and the field names differ: `files = { 'src/parser.c', 'src/scanner.c' }` replaces `queries`, and the parser config is `require('nvim-treesitter.parsers').get_parser_configs().tdl`, which also takes a `filetype`.
That branch installs no queries for a custom parser, so `tree-sitter/queries/highlights.scm` goes in `queries/tdl/highlights.scm` somewhere on the runtimepath.

Both sources are compiled because the grammar has an external scanner.

### VS Code

`nix build .#vscode-tdl` builds the extension in [editors/vscode](editors/vscode), and `overlays.default` exposes it as `pkgs.vscode-tdl`.
`programs.tdl.vscode.enable` in the home-manager module puts it in a VS Code profile; by hand, add it to `vscode-with-extensions` or to home-manager's `programs.vscode.profiles.<name>.extensions`.
`make vscode-install` packages it as a `.vsix` and hands it to `code --install-extension`, which is the route to use when iterating on the colors.

## Support matrix

What each part of the language reaches today.
`Front end` is the lexer, parser, and `tdl fmt`; `IR` is `tdl ir`, the resolved model a backend consumes.

| Construct | Front end | IR |
| --- | --- | --- |
| `package`, `import` | Yes | Yes, resolved across packages without inlining |
| `primitive` | Yes | Yes |
| `alias` | Yes | Yes |
| `type` (newtype chains) | Yes | Yes, constraints accumulate down the chain |
| `value`, `entity` | Yes | Yes |
| `mixin`, `include` | Yes | Yes, expanded |
| `enum`, variants with fields | Yes | Yes |
| `class`, functional dependencies, associated types | Yes | Yes |
| `instance`, including conditional instances | Yes | Yes, satisfaction answers for both |
| Type parameters and kinds | Yes | Yes, parameters stay parameters |
| Collection and option sugar (`[T]`, `{T}`, `{K -> V}`, `T?`, `T \| null`) | Yes | Yes, lowered to whatever the prelude declares |
| `where` constraints | Yes | Yes, open set: standard names checked, others passed through |
| Field defaults | Yes | Yes, resolved against the field's type |
| `deprecated` | Yes | Yes |
| `target` blocks | Yes | Partial: resolved and attached, dependency blocks not merged |
| `unit` | Yes | No: declarations pass through unlowered, unit arguments are an error |

`tdl fmt` drops ordinary `//` comments; `///` doc comments survive, since they attach to the declaration that follows.

Backends:

| Backend | Kind | Status |
| --- | --- | --- |
| `debug` | Built in, also shipped as `tdl-gen-debug` | Prints the model it was given |
| Anything else | `tdl-gen-<name>` on PATH | The protocol is stable, none written |

## Releases

Versions come from [release-please](https://github.com/googleapis/release-please): a release pull request accumulates conventional commits and, when merged, tags a release and writes `CHANGELOG.md`.
Nothing about a version is edited by hand; `toolVersion` and the Nix package version carry annotations that the release PR rewrites.

## Development

```shell
command make build   # nix build .#
command make test    # go test ./...
command make lint    # nix flake check + golangci-lint + buf + markdownlint
command make fmt     # nix fmt + buf format
```

`go build ./...` and `go test ./...` work directly for anyone not using Nix.

## Design philosophy

- The language core is small. Almost everything that looks like a type system is library code written in TDL and shipped in a replaceable prelude.
- Identity is first class, and the model is pure: a `.tdl` file describes the domain, and everything a backend needs lives in a `target` block.
- Constraints are syntax, not semantics. The compiler parses and resolves them; backends decide what they mean.
- A small, strict grammar with a hand-written lexer and parser. No parser generator, no YAML or JSON stand-in syntax.
- One canonical Go implementation. The spec and the `testdata/conformance` and `testdata/invalid` corpora are the contract another implementation would satisfy, which is why they are plain text rather than Go tests.
