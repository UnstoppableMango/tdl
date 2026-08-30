# Backlog

Work that is wanted but not scheduled.
Anything with a plan of its own lives in [design/](design/) instead; this is the list of things that have not earned one yet.

Ordered roughly by what unblocks what, not by priority.

## tdl fmt as a treefmt formatter

`nix fmt` formats everything in the repository except `.tdl` files, which are excluded with a note pointing here.
`tdl fmt` already produces canonical output and is idempotent, so this is a custom formatter entry in `flake.nix` and a way to run the locally built binary rather than a released one.

The awkward part is bootstrapping: formatting the repository would depend on building the thing the repository is.

## Tree-sitter grammar

A tree-sitter grammar gives syntax highlighting, structural selection, and folding to every editor that speaks it, which is most of them now.

It is a second parser, and a second parser is a second thing to keep in step with `docs/grammar.ebnf`.
The conformance corpus is the obvious way to hold them together: a tree-sitter grammar that parses `testdata/conformance/*/source.tdl` and rejects `testdata/invalid/*/source.tdl` agrees with the reference implementation by construction.

This is the dependency for most of the editor work below.

## Language server

`tdl lsp` is already described in [design/workflow.md](design/workflow.md) as the editor-facing half of the inner loop.

The pieces exist: the parser reports every error in one pass with positions, and `internal/sema` resolves names and records where each declaration came from.
What is missing is the protocol layer and incremental reparsing.

Diagnostics, go-to-definition, and hover are the first three features worth having, in that order.
Completion needs scope information the resolver already computes.

## Editor support

Each of these wants the tree-sitter grammar for highlighting and the language server for everything else, so neither is worth starting before those exist.

- **Vim and Neovim.** Filetype detection, a tree-sitter parser registration, and an `lspconfig` entry. The smallest of the four.
- **VS Code.** An extension bundling the language server, plus a grammar for highlighting before the server starts.
- **JetBrains.** A plugin. The platform has its own PSI model, so this is the most work of the four; the LSP API narrows it.
- **Emacs.** A major mode deriving from `prog-mode`, `treesit` integration, and an `eglot` entry.

## MCP server

An MCP server would let an agent read a resolved model rather than the source text: ask what an entity's fields are, what satisfies a class, what a target block maps.

`tdl ir --format json` already emits the whole model, so a first version is a thin wrapper over the existing lowering.
The interesting question is what tools it should offer beyond "give me the model", and that is worth answering after there is a backend to compare against.
