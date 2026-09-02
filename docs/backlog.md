# Backlog

Work that is wanted but not scheduled.
Anything with a plan of its own lives in [design/](design/) instead; this is the list of things that have not earned one yet.

Ordered roughly by what unblocks what, not by priority.

## tdl fmt as a treefmt formatter

`nix fmt` formats everything in the repository except `.tdl` files, which are excluded with a note pointing here.
`tdl fmt` already produces canonical output and is idempotent, so this is a custom formatter entry in `flake.nix` and a way to run the locally built binary rather than a released one.

The awkward part is bootstrapping: formatting the repository would depend on building the thing the repository is.

## Anonymous union types

A field wants to say `A | B` without a declaration standing behind it.
`enum` covers the nominal case, since a variant may carry fields, but every alternative has to be named and declared in one place.

The syntax is already half there.
`TypeRef = CoreType [ "?" ] [ "|" "null" ]` admits a single `|`, and `T | null` lowers to the prelude's `Nullable<T>`, so `|` is sugar for a named generic rather than a form of its own.
The general case is that same rule with the right-hand side opened up.

The open question is arity.
`A | B | C` wants a variadic `Union<A, B, C>`, and kinds are `type`, `unit`, and arrows between them, with nothing variadic.
Nesting instead gives `Either<A, Either<B, C>>`, which makes one written form into two types depending on how it associates, and makes `A | B` and `B | A` distinct.
Answering that decides whether this needs a kind-system change, which is what would earn it a plan of its own.

Downstream of the answer: whether a union may appear anywhere a `TypeRef` may, what a backend receives in `ir`, and whether the recursion rules in [spec.md](spec.md) treat reaching yourself through a union the way they treat a collection or an optional.

## Language server

`tdl lsp` is already described in [design/workflow.md](design/workflow.md) as the editor-facing half of the inner loop.

The pieces exist: the parser reports every error in one pass with positions, and `internal/sema` resolves names and records where each declaration came from.
What is missing is the protocol layer and incremental reparsing.

Diagnostics, go-to-definition, and hover are the first three features worth having, in that order.
Completion needs scope information the resolver already computes.

## Editor support

Each of these wants the tree-sitter grammar for highlighting and the language server for everything else, so neither is worth starting before those exist.
The grammar has a plan of its own in [design/treesitter-plan.md](design/treesitter-plan.md).

- **Vim and Neovim.** Filetype detection, a tree-sitter parser registration, and an `lspconfig` entry. The smallest of the four.
- **VS Code.** An extension bundling the language server, plus a grammar for highlighting before the server starts.
- **JetBrains.** A plugin. The platform has its own PSI model, so this is the most work of the four; the LSP API narrows it.
- **Emacs.** A major mode deriving from `prog-mode`, `treesit` integration, and an `eglot` entry.

## MCP server

An MCP server would let an agent read a resolved model rather than the source text: ask what an entity's fields are, what satisfies a class, what a target block maps.

`tdl ir --format json` already emits the whole model, so a first version is a thin wrapper over the existing lowering.
The interesting question is what tools it should offer beyond "give me the model", and that is worth answering after there is a backend to compare against.
