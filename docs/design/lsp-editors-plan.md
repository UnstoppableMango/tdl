# Taking the language server to editors

An implementation plan continuing [lsp-plan.md](lsp-plan.md), whose four phases are done.
Phases are ordered by priority and then by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

`tdl lsp` serves diagnostics, definition, hover, formatting, and an outline, and no editor starts it.
The VS Code extension contributes a language and a grammar and nothing else, and there is no Neovim, Helix, Emacs, or Zed configuration anywhere in the repository.
Every feature so far reaches only someone who wires the server up by hand, so the clients come first and the features after them.

Rich editor support is a priority for this project, which is why the clients are five phases rather than one: each editor has its own way of finding a server, and a plan that covered only VS Code would leave the other four where they are.

## Principles

**The server does the work.**
A client starts `tdl lsp` and forwards what the editor asks.
Anything a client would compute is something four other clients would compute differently.

**The server is `tdl` on `PATH` unless configured otherwise.**
Every client takes a path to the executable and defaults to `tdl`, so the version an editor runs is the version a terminal runs.
Where home-manager configures an editor, the module sets that path to `programs.tdl.package`, so the two cannot drift.

**Every client is verified by opening a conformance file.**
Done means a file with an undefined type underlines it, hovering a name shows its declaration, and formatting rewrites a messy file, in that editor.
The server's behaviour is tested in `internal/lsp`; what a phase here proves is that the editor reached it.

## Phase 1: the VS Code client

`editors/vscode/` gains an entry point that starts `tdl lsp` through `vscode-languageclient` over stdio, and a `tdl.server.path` setting defaulting to `tdl`.
`package.json` gains `main`, the dependency, and an `engines.vscode` raised to what `vscode-languageclient` requires.
Activation is implicit, since VS Code activates an extension for a language it contributes.

The entry point is bundled with esbuild into one file, because an extension that ships `node_modules` is slower to load and larger to install.
`nix/vscode-extension.nix` builds the bundle with `buildNpmPackage` from a committed `package-lock.json`, then hands the directory to `buildVscodeExtension` as it does today.
`editors/vscode/install.sh` builds the bundle before packaging, so `make vscode-install` keeps working.

The home-manager module sets `tdl.server.path` in each profile's user settings when both `programs.tdl.vscode.enable` and the extension are on.

When the executable is missing, the extension says so once with the path it tried, and highlighting keeps working, since the grammar needs no server.

Done when `nix build .#vscode-tdl` produces an extension that shows a diagnostic, a hover, and a formatting edit on a conformance file, and a missing `tdl` produces one message rather than a crash.

## Phase 2: Neovim

`README.md`'s Neovim section gains the server, using the configuration API Neovim 0.11 added:

```lua
vim.lsp.config('tdl', { cmd = { 'tdl', 'lsp' }, filetypes = { 'tdl' }, root_markers = { '.git' } })
vim.lsp.enable('tdl')
```

`vim.filetype.add` for `*.tdl` is already in the section, and the server needs nothing more.

Adding `tdl` to `nvim-lspconfig` is distribution and waits, for the reason the Marketplace does in [editors-plan.md](editors-plan.md).

Done when pasting the section into a configuration that has never seen TDL shows a diagnostic and a hover on a conformance file, and `:checkhealth vim.lsp` lists the client attached.

## Phase 3: Helix

A `languages.toml` section in `README.md`: a `[[language]]` entry for `tdl` naming the server and the comment token, a `[language-server.tdl]` entry running `tdl lsp`, and a `[[grammar]]` entry pointing at this repository with `subpath = "tree-sitter"`.

Helix reads highlight queries from its runtime directory rather than from the grammar, so the section copies `tree-sitter/queries/highlights.scm` into `runtime/queries/tdl/`.
Helix's capture names mostly match the `nvim-treesitter` names the query uses; the ones that do not are listed in the section rather than forked into a second query file.

The home-manager module does not configure Helix: `programs.helix.languages` takes the same TOML as a Nix value, and the README shows it.

Done when `hx --health tdl` reports the server and the grammar, and a conformance file shows a diagnostic and a hover.

## Phase 4: Emacs

`editors/emacs/tdl-mode.el`: a major mode deriving from `prog-mode` with the comment syntax, `auto-mode-alist` for `*.tdl`, and an entry in `eglot-server-programs` running `tdl lsp`.
Eglot is built into Emacs 29, so the mode needs no dependency.

The mode's font-lock rules cover keywords, comments, and literals, with the keyword list read from `lex.Keywords()` by a test so the two cannot drift.
Colouring names by what they are is the tree-sitter grammar's job, and `tdl-ts-mode`, reading it through `treesit`, is a later addition beside this mode rather than a replacement.

Done when loading the file in `emacs -Q` colours a conformance file's keywords, and running `eglot` on it shows a diagnostic and a hover.

## Phase 5: Zed

Zed runs a language server only through an extension, and an extension naming a server is Rust compiled to WebAssembly: `language_server_command` finds `tdl` on the worktree's `PATH` and returns `tdl lsp`.

It depends on phases 4 and 5 of [editors-plan.md](editors-plan.md), the grammar repository and the Zed extension carrying highlighting, and adds the server to that extension rather than making a second one.

Done when the extension loaded as a dev extension shows a diagnostic and a hover on a conformance file.

## Phase 6: target-block directives

`tdl gen` checks each directive against the `DirectiveSpec` its backend declares, with `gen.CheckDirectives`, and nothing checks them while editing.

The server runs the same check for every target block naming a built-in backend, reading each one's `Describe` in process through `gen.Builtin`.
An undeclared directive publishes as a warning and a wrong argument count or kind as an error, which is what `tdl gen` reports.
A backend found on `PATH` is not started to ask, since starting an executable because a file was opened is a surprise.

This is the first diagnostic the server publishes that is not an error, so `publish` gains severities here.

Done when a misspelled directive in a `target go` block is underlined as it is typed, and the corpus still publishes nothing.

## Phase 7: completion

Completion needs the names in scope at a cursor, which lowering computes and discards, and it is asked while the text is being typed and so usually does not parse.

Two changes carry it.
`internal/sema` gains a way to list the bindings visible from a declaration: the file scope, the prelude beneath it, and the declaration's type parameters, which is the ladder `lookup` already climbs.
The server keeps the last snapshot that lowered, so a half-typed field completes against the model as it was a keystroke ago, with the lexer deciding the position from the tokens before the cursor.

What a position offers:

- A type position: declarations, type parameters, and names a `_` import merged in; after `alias.`, what that import exports.
- After `:` in a declaration head and after `requires`: classes.
- After `include`: mixins.
- Inside `where { }`: the standard constraint names from `internal/sema/constraint.go`, with their arity.
- Inside a target block: declaration names, then fields and variants after `.`, and the backend's directive names from phase 6, each with its argument kinds.

Each item carries its kind and, resolved lazily, the hover text phase 3 renders.

Done when typing a field's type offers the model's declarations and the prelude's, `include` offers only mixins, and a directive position in a `target go` block offers `tag` and `key`.

## Phase 8: references, rename, and highlight

The index records every resolution, and these three want every occurrence.
`internal/sema` records the ones it misses today: a declaration's own name, a type parameter's, `include`, unit names in a unit expression, constructors in a default, and each segment of a target path, which the parser keeps as one string and so has to give positions per segment.

`textDocument/references` is the records sharing a target.
`textDocument/documentHighlight` is the same set within one file.
`textDocument/rename` edits every one of them, after `prepareRename` refuses a cursor on the prelude, a keyword, or a qualified name into a dependency, and the new name is checked against the lexer's identifier rules and the reserved words.
A rename that would collide with a name already in scope is refused rather than applied.

Done when renaming a declaration used in a field, an `include`, a target path, and a file importing it with `_` edits all four and the corpus still publishes nothing.

## Phase 9: semantic tokens

Highlighting from the model rather than from a grammar: a name is coloured by what it resolved to, so a mixin, a class, an entity, and a type parameter can differ, and a reference to a deprecated declaration carries the `deprecated` modifier that editors strike through.

The token types follow the captures `tree-sitter/queries/highlights.scm` already chose, so an editor using both sees the same colours.
Keywords and literals come from the lexer, and names come from the index phase 8 completes.

Done when a reference to a deprecated declaration is struck through in VS Code and in Neovim.

## Phase 10: structured diagnostics and quick fixes

`sema.Diagnostic` is a position and a message, so an editor cannot tell one kind of problem from another.
It gains a code, a severity, and related locations: the first declaration in a duplicate, which today is text in the message.

Codes are what code actions key on.
The first fixes are the ones the model can answer: an undefined name offers the declared names closest to it, and a name another file declares offers the `_` import that brings it in.

Done when an undefined type offers its likely spelling as a quick fix, and applying it clears the diagnostic.

## Not in this plan

- **Publishing.** The Marketplace, `nvim-lspconfig`, MELPA, and the Zed registry are distribution, as [editors-plan.md](editors-plan.md) argues.
- **JetBrains.** [backlog.md](../backlog.md) has it, and its LSP API would reuse everything here, but it is the most work of any editor.
- **Folding, selection ranges, workspace symbols, and inlay hints.** Each is small and additive once completion and the full index exist, and none is what a person misses first.
- **The prelude under a `tdl:` URI.** [lsp.md](lsp.md) argues hover is the answer.
