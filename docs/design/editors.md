# Editor support

Design document.
Nothing here is built.
The tree-sitter grammar it all leans on is built already, and [treesitter.md](treesitter.md) describes it.

The goal is syntax highlighting in Neovim, VS Code, Zed, and on GitHub, for TDL files.
Everything past highlighting wants the language server, which is separate work and is not in this document.

[editors-plan.md](editors-plan.md) tracks which phase has landed.

## What each editor actually reads

The tree-sitter grammar is not universal.
Two of the four targets cannot use it, and knowing which is the whole shape of this document.

| Target | Reads | Uses `tree-sitter/` |
| --- | --- | --- |
| Neovim | tree-sitter, through nvim-treesitter | directly |
| Zed | tree-sitter, through a Zed extension | from a repository of its own |
| VS Code | a TextMate grammar | no |
| GitHub | a TextMate grammar, through Linguist | no |

VS Code's highlighting is TextMate and has been since the beginning; every tree-sitter option for it is a third-party extension going through the semantic token API, which is a different feature with different behaviour.
Linguist requires a TextMate grammar and does not accept a tree-sitter one.

So there is a second grammar to write, and a second grammar is a second thing to keep in step with [grammar.ebnf](../grammar.ebnf).
That is the problem [treesitter.md](treesitter.md) already answered once.

## The TextMate grammar is derived too

`internal/textmate` reads the same annotated grammar `internal/treesitter` reads, and writes `editors/vscode/syntaxes/tdl.tmLanguage.json`.
`tools/textmate` is the `main` that runs it, beside `tools/treesitter`.

TextMate is regular expressions over lines and cannot express the grammar's structure, so the derived file is a lexical approximation where the tree-sitter one is exact.
That is the right trade rather than a compromise: the lexical layer is the part `lex/table.go` already states for a program.
`Keywords`, `Punctuation`, and the six `Pattern` constants are the whole input, and each is read from the tables `Lexer.Next` dispatches on or held to them by a test.

What it colours is therefore what cannot drift: keywords, punctuation, the literals, comments, and doc comments.
What it does not colour is anything needing a parse, and a declaration name reads as an identifier rather than as a type.

Deriving it is what makes that acceptable.
A hand-written approximation rots the first time a keyword is added; a derived one fails the regeneration check instead.

The generated file is committed and checked the way `tree-sitter/grammar.js` is, by a Go test that rewrites it under `-update`.

## Neovim

The grammar needs no packaging.
nvim-treesitter's `install_info` takes a `location` for a parser in a subdirectory and a `queries` for the query files beside it, which is exactly this repository's shape.

What is needed is documentation: a README section with `vim.filetype.add` for `*.tdl`, the parser registration, and `vim.treesitter.language.register`.
A user pastes it into their configuration and has highlighting.

A plugin repository would remove the pasting and is worth having later.
It is not worth having first, because the snippet is the thing the plugin would contain.

## VS Code

An extension in `editors/vscode/`, holding `package.json`, the language contribution, and the derived `syntaxes/tdl.tmLanguage.json`.

It is installed through nix rather than the Marketplace.
`vscode-utils.buildVscodeExtension` in nixpkgs builds an extension from a local source, and the flake exposes it as `packages.vscode-tdl` and as `pkgs.vscode-tdl` through `overlays.default`, so `programs.tdl.vscode.enable` in the home-manager module can put it in a VS Code profile, and a user outside that module adds it to `vscode-with-extensions` or to `programs.vscode.profiles.<name>.extensions`.

That is the whole distribution story for now.
Publishing to the Marketplace is an account, a token, and a release job, and none of it changes the extension.
It waits until someone outside a nix configuration wants this.

The extension declares the language, the file extension, and the grammar, and nothing else.
Bundling the language server is what the extension is eventually for, and adding it later changes no file this phase writes.

## Zed

An extension with `extension.toml` and `languages/tdl/config.toml`, using the tree-sitter grammar and the same `highlights.scm` this repository already has.

Zed loads a grammar from `repository` and `rev` and has no key for a subdirectory, so it cannot be pointed at `tree-sitter/` in this repository.
This is the one target that forces a structural change, and it is the reason for the next section.

## Splitting the grammar

`tree-sitter-tdl`, a repository of its own, generated from this one rather than moved out of it.

Moving it out is the obvious reading and the wrong one.
The derivation's whole value is that a production reaching `docs/grammar.ebnf` and not the derived parser is a diff that fails one CI job, and that job cannot span two repositories.
`docs/design/treesitter.md` says a repository per grammar can be pointed at a subdirectory instead; that was written before Zed's constraint was known, and the sentence is wrong.

So the source of truth stays here and the split repository is an artifact.
A release job pushes `tree-sitter/` into it: `grammar.js`, `src/`, `queries/`, and `tree-sitter.json`, plus a `package.json` this repository does not need.
The mirror is tagged with the same version, and Zed pins a `rev`.

This costs one job and keeps every check where it is.
It also gives the grammar a home an editor that expects one can use, which is the thing Zed asked for and will not be the last to.

## GitHub

Through Linguist, when the numbers are there, and not before.

Linguist wants a TextMate grammar vendored into its tree, and at least 2000 files per extension indexed in the last year across many repositories.
The grammar this document derives satisfies the first.
The second is adoption, and adoption is not something to build.

There is a way to have colour sooner.
`*.tdl linguist-language=X` in a repository's `.gitattributes` gives that repository's TDL files the highlighting of an existing language, and Kotlin, Swift, and Thrift are each close enough to be legible.

It is not worth it.
The override classifies as well as colours, so every repository doing it reports itself as written in a language it is not, in the language bar and in the API behind it.
A language young enough to need the trick is exactly the one that cannot afford to be counted as something else.

So GitHub waits.
The derived grammar means the waiting is on usage rather than on work, which is the right thing to be blocked on.

## Not in this document

- The language server. `tdl lsp` is described in [workflow.md](workflow.md), and every target above wants it more than it wants highlighting. Nothing here blocks it, and it changes none of these files.
- JetBrains and Emacs. Both are in [backlog.md](../backlog.md). JetBrains has its own PSI model and is the largest of the six; Emacs wants `treesit` and a major mode, and is close to Neovim in shape.
- Marketplace and registry publishing. A distribution question rather than a correctness one, for VS Code, for Zed, and for npm.
- Anything but `highlights.scm`. `brackets.scm`, `outline.scm`, `indents.scm`, and injections are each additive once a target reads the first one.
