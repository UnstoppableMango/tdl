# Backlog

Work that is wanted but not scheduled.
Anything with a plan lives in [design/](design/) instead.

## tdl fmt as a treefmt formatter

`nix fmt` excludes `.tdl` files.
Adding `tdl fmt` as a custom formatter in `flake.nix` needs the locally built binary, so formatting the repository would depend on building it.

## Anonymous union types

A field wants `A | B` without a declaration; `enum` requires every alternative to be named and declared.
`TypeRef` already admits `T | null` as sugar for `Nullable<T>`, so the general case opens up the right-hand side.

The open question is arity: `A | B | C` wants a variadic `Union`, and kinds have nothing variadic, while nesting into `Either` makes association and order significant.
The answer decides whether this needs a kind-system change, and with it what `ir` carries and how the recursion rules treat a union.

## An exponent on a parenthesized unit term

`docs/grammar.ebnf` has `UnitTerm = identifier [ "^" int_lit ] | "(" UnitExpr ")" .`, but `parser.parseUnitTerm` accepts `(kg*m)^2` and lowers it to `kg^2*m^2`.
The parser is right; the fix is `UnitTerm = ( identifier | "(" UnitExpr ")" ) [ "^" int_lit ] .`, a sentence in the spec's units section, and a regenerated tree-sitter grammar.

## JetBrains

A plugin over the platform's PSI model, narrowed by its LSP API.
The other editors are planned in [design/editors-plan.md](design/editors-plan.md) and [design/lsp-editors-plan.md](design/lsp-editors-plan.md).

## MCP server

Lets an agent query a resolved model: an entity's fields, what satisfies a class, what a target block maps.
`tdl ir --format json` already emits the model, so a first version is a thin wrapper; which tools to offer beyond that is the open question.
