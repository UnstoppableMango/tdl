# tdl

[![CI](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml/badge.svg)](https://github.com/UnstoppableMango/tdl/actions/workflows/ci.yml)
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
`union` is reserved and unimplemented; nothing else in [grammar.ebnf](docs/grammar.ebnf) is missing.

**The middle is most of the way there.** `tdl ir` prints a resolved model:

- Names resolve, with scopes, shadowing, and the spec's recursion rules.
- Sugar lowers to prelude types, and the prelude is TDL source rather than something built in, so `[T]` means whatever the loaded prelude says `List` is.
- Imports resolve across packages without inlining the dependency.
- Mixins expand, and the class satisfaction index answers both for declarations and for types made to satisfy a class by a conditional instance.
- Constraints accumulate down newtype chains, and defaults resolve against their field's type.
- Target directives resolve against the model and attach to the nodes they apply to.

**The back end runs, with nothing behind it yet.** `tdl gen` resolves a target's backend to a built-in or to `tdl-gen-<name>` on PATH, sends it the resolved model, and writes back the files it returns.
The only backend is `debug`, which prints what it was given, so nothing generates real code yet.
Units, and merging a dependency's target blocks, are the two pieces of resolution still outstanding.

The design is settled and written down:

| Document | What it covers |
| --- | --- |
| [spec.md](docs/spec.md) | The language. Canonical. |
| [grammar.ebnf](docs/grammar.ebnf) | The formal grammar. |
| [design/parser-plan.md](docs/design/parser-plan.md) | Rewriting the lexer and parser to match. Done. |
| [design/ir.md](docs/design/ir.md) | The resolved model backends consume. |
| [design/ir-plan.md](docs/design/ir-plan.md) | Implementing it. Phases 1 to 8 of 10 done. |
| [design/plugins.md](docs/design/plugins.md) | The backend plugin protocol. |
| [design/plugins-plan.md](docs/design/plugins-plan.md) | Implementing it. Phases 1 to 8 of 8 done. |
| [design/workflow.md](docs/design/workflow.md) | What a model author does with all of it. |
| [backlog.md](docs/backlog.md) | Wanted, unscheduled: tree-sitter, an LSP, editor support, an MCP server. |

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
| `union` | No: reserved | No |

`tdl fmt` drops ordinary `//` comments; `///` doc comments survive, since they attach to the declaration that follows.

Backends:

| Backend | Kind | Status |
| --- | --- | --- |
| `debug` | Built in, also shipped as `tdl-gen-debug` | Prints the model it was given |
| Anything else | `tdl-gen-<name>` on PATH | The protocol is stable, none written |

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

## Usage

```shell
tdl check ./types.tdl    # parse and report syntax errors
tdl fmt ./types.tdl      # print canonical formatting; -w to write in place
tdl ast ./types.tdl      # print the parse tree
tdl gen ./types.tdl      # run every target block; --target narrows, -o overrides
                         # --verify checks, --clean empties first, --watch reruns
tdl ir ./types.tdl       # print the resolved model; --format json for the plugin view
tdl tokens ./types.tdl   # print the token stream
tdl version              # tool and spec versions
```

### Playground

`tdl play` watches a file and re-renders it on every save.

```shell
tdl play                              # scratch.tdl, created from a template if missing
tdl play ./types.tdl --views all      # source, fmt, ast, tokens, stats
tdl play ./types.tdl --views fmt      # one pane
tdl play ./types.tdl --once           # render and exit
```

Views are `source`, `fmt`, `ast`, `tokens`, `stats`, or `all`; the default is `fmt,ast`.
Parse errors render below the panes with a caret at the reported column.

[`examples/`](examples/README.md) holds files to start from: the same domain modelled flat and nested, plus collections and target blocks.

## Releases

Versions come from [release-please](https://github.com/googleapis/release-please): a release pull request accumulates conventional commits and, when merged, tags a release and writes `CHANGELOG.md`.

Nothing about a version is edited by hand. `toolVersion` and the Nix package version carry annotations that the release PR rewrites.

## Development

```shell
command make build   # nix build .#
command make test    # go test ./...
command make lint    # nix flake check + buf lint
command make fmt     # nix fmt
```

`go build ./...` and `go test ./...` work directly for anyone not using Nix.

## Design philosophy

- The language core is small. Almost everything that looks like a type system is library code written in TDL and shipped in a replaceable prelude.
- Identity is first class, and the model is pure: a `.tdl` file describes the domain, and everything a backend needs lives in a `target` block.
- Constraints are syntax, not semantics. The compiler parses and resolves them; backends decide what they mean.
- A small, strict grammar with a hand-written lexer and parser. No parser generator, no YAML or JSON stand-in syntax.
- One canonical Go implementation. The spec and the `testdata/conformance` and `testdata/invalid` corpora are the contract another implementation would satisfy, which is why they are plain text rather than Go tests.
