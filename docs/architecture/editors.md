# Editor support

Design document.
The goal is syntax highlighting in Neovim, VS Code, Zed, bat, and on GitHub.
Neovim, the TextMate grammar, VS Code, and bat are built; the grammar mirror, Zed, and GitHub are not.
[#1204](https://github.com/UnstoppableMango/tdl/issues/1204) tracks the remaining work, [treesitter.md](treesitter.md) describes the tree-sitter grammar, and [lsp.md](lsp.md) describes the language server.

## What each editor reads

| Target | Reads | Uses `tree-sitter/` |
| --- | --- | --- |
| Neovim | tree-sitter, through nvim-treesitter | directly |
| Zed | tree-sitter, through a Zed extension | from a repository of its own |
| VS Code | a TextMate grammar | no |
| bat | a sublime-syntax, through syntect | no |
| GitHub | a TextMate grammar, through Linguist | no |

VS Code highlights with TextMate, and Linguist accepts only a TextMate grammar, so there is a second grammar to keep in step with [grammar.ebnf](../grammar.ebnf).

## The TextMate grammar is derived too

`internal/textmate` reads the same annotated grammar `internal/treesitter` reads and writes `editors/vscode/syntaxes/tdl.tmLanguage.json`; `tools/textmate` runs it.
A hand-written grammar would go stale the first time a keyword is added, and a derived one fails the regeneration check instead.

TextMate is regular expressions over lines, so the output is a lexical approximation.
Its input is `lex.Keywords`, `lex.Punctuation`, and the `*Pattern` constants in `lex/table.go`, plus the names a neighboring token gives away ([editors-plan.md](https://github.com/UnstoppableMango/tdl/blob/fc0682d/docs/design/editors-plan.md) phase 2 lists them).

The generated file is committed; `go test ./internal/textmate` compares it and `make textmate` regenerates it.

## bat

bat reads only sublime-syntax files, so `internal/textmate` writes the same rules a second time as `editors/bat/tdl.sublime-syntax`; `make textmate` writes both, and the test compares both.
The two formats match the same way, earliest in the line and then first listed, so the second is a translation and not a second grammar.
Sublime Text reads the same file.

The `tdl` package installs it under `share/bat/syntaxes/`, and `programs.tdl.bat.enable` in the home-manager module, on whenever `programs.bat.enable` is, hands it to `programs.bat.syntaxes`, which rebuilds bat's cache.
The README's demo highlights with it too, so the GIF shows what a terminal shows; tree-sitter's parse-based colors were the trade.
Adding TDL to bat itself waits, like GitHub: bat takes a syntax as a submodule of a package repository of its own.

## Neovim

nvim-treesitter's `install_info` takes a `location` for a parser in a subdirectory and a `queries` for the query files beside it, which is this repository's layout.
The README carries a configuration snippet, so no plugin repository is needed yet.

## VS Code

An extension in `editors/vscode/` contributing the language, the file extension, the derived grammar, and a client for `tdl lsp`.

It is installed through nix: `packages.vscode-tdl`, also `pkgs.vscode-tdl` through `overlays.default`, which `programs.tdl.vscode.enable` in the home-manager module puts in a VS Code profile.
Marketplace publishing (an account, a token, and a release job) waits until someone outside a nix configuration wants it.

## Zed

An extension with `extension.toml` and `languages/tdl/config.toml`, using the tree-sitter grammar and `highlights.scm`.
Zed loads a grammar from `repository` and `rev` with no key for a subdirectory, so it needs a repository holding only the grammar.

## Splitting the grammar

`tree-sitter-tdl` is a mirror generated from this repository, not a move out of it.
The regeneration check is one CI job here, and it cannot span two repositories.

A release job pushes `tree-sitter/` into the mirror: `grammar.js`, `src/`, `queries/`, and `tree-sitter.json`, plus a `package.json` for grammar consumers.
The mirror is tagged with the same version, and Zed pins a `rev`.

## GitHub

Through Linguist, once TDL has the usage Linguist requires (at least 2000 files per extension indexed in the last year across many repositories).
The derived TextMate grammar meets the other requirement.

A `.gitattributes` override such as `*.tdl linguist-language=Kotlin` would color files sooner, but it also reclassifies them, so every repository using it would report itself as written in another language.
GitHub waits.

## Not in this document

- JetBrains: [#1310](https://github.com/UnstoppableMango/tdl/issues/1310). Emacs: [#1216](https://github.com/UnstoppableMango/tdl/issues/1216).
- Marketplace and registry publishing, for VS Code, Zed, and npm.
- Query files other than `highlights.scm` (`brackets.scm`, `outline.scm`, `indents.scm`, injections), each additive.
