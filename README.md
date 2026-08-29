# tdl

TDL is a language for describing domain models.
It says what things are, what identifies them, how they relate, and what values they may hold, and compiles them into equivalent definitions in other structured formats.
It describes no behavior and has no expressions, control flow, or runtime.

This repository owns the canonical [language specification](docs/spec.md) and its reference implementation, written in Go.

## Status

Early and incomplete.

The lexer and parser read the whole grammar, and `tdl check`, `tdl fmt`, `tdl ast`, and `tdl tokens` work across it.
`union` is reserved and unimplemented; nothing else in [grammar.ebnf](docs/grammar.ebnf) is missing.

The design is settled and written down:

| Document | What it covers |
| --- | --- |
| [spec.md](docs/spec.md) | The language. Canonical. |
| [grammar.ebnf](docs/grammar.ebnf) | The formal grammar. |
| [design/parser-plan.md](docs/design/parser-plan.md) | Rewriting the lexer and parser to match. Done. |
| [design/ir.md](docs/design/ir.md) | The resolved model backends consume. |
| [design/ir-plan.md](docs/design/ir-plan.md) | Implementing it. Phases 1 to 3 of 8 done. |
| [design/plugins.md](docs/design/plugins.md) | The backend plugin protocol. |
| [design/workflow.md](docs/design/workflow.md) | What a model author does with all of it. |

Semantic resolution has started: the `ir` schema, the declaration table, the interned type table, name resolution, and the spec's recursion rules are in, so a single-package model lowers with its sugar resolved to prelude types and its names bound. `tdl ir` prints the result. Imports, classes, constraints, and targets are still to come, and there are no code-generation backends.

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

## Development

```shell
command make build   # nix build .#
command make test    # go test ./...
command make lint    # nix flake check
command make fmt     # nix fmt
```

`go build ./...` and `go test ./...` work directly for anyone not using Nix.

## Design philosophy

- The language core is small. Almost everything that looks like a type system is library code written in TDL and shipped in a replaceable prelude.
- Identity is first class, and the model is pure: a `.tdl` file describes the domain, and everything a backend needs lives in a `target` block.
- Constraints are syntax, not semantics. The compiler parses and resolves them; backends decide what they mean.
- A small, strict grammar with a hand-written lexer and parser. No parser generator, no YAML or JSON stand-in syntax.
- One canonical Go implementation. The spec and the `testdata/conformance` and `testdata/invalid` corpora are the contract another implementation would satisfy, which is why they are plain text rather than Go tests.
