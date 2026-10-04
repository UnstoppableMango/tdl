# Taking the language server to editors

An implementation plan continuing [lsp-plan.md](lsp-plan.md).
Phases are ordered by priority and then by dependency, and each states what makes it done.
Phase 1 is done.

## Scope

Clients for VS Code, Neovim, Helix, Emacs, and Zed first, since each editor finds a server differently, then the server features after them.

## Principles

**The server does the work.**
A client starts `tdl lsp` and forwards requests; anything a client computed, the other clients would compute differently.

**The server is `tdl` on `PATH` unless configured otherwise.**
Every client takes a path to the executable and defaults to `tdl`.
Where nix builds the client, the default is the `tdl` it was built against, and no user setting is written.

**Every client is verified in its editor on a conformance file.**
Done means an undefined type is underlined, hovering a name shows its declaration, and formatting rewrites a messy file.
The server's behavior is tested in `internal/lsp`; a phase here proves the editor reached it.

## Phase 1: the VS Code client (done)

`editors/vscode/src/extension.ts` starts `tdl lsp` through `vscode-languageclient` over stdio, reading `tdl.server.path` (default `tdl`).
esbuild bundles it into `dist/extension.js`.
`nix/vscode-extension.nix` builds the bundle with `buildNpmPackage` and rewrites the setting's default to the built `tdl` with `jq`.
A missing executable produces one message naming the path tried, and highlighting keeps working.

Done when `nix build .#vscode-tdl` produces an extension that shows a diagnostic, a hover, and a formatting edit on a conformance file, and a missing `tdl` produces one message rather than a crash.

## Phase 2: Neovim

`README.md`'s Neovim section gains the server, using Neovim 0.11's configuration API:

```lua
vim.lsp.config('tdl', { cmd = { 'tdl', 'lsp' }, filetypes = { 'tdl' }, root_markers = { '.git' } })
vim.lsp.enable('tdl')
```

The section already has `vim.filetype.add` for `*.tdl`.
Adding `tdl` to `nvim-lspconfig` is distribution and waits.

Done when the section pasted into a fresh configuration shows a diagnostic and a hover on a conformance file, and `:checkhealth vim.lsp` lists the client attached.

## Phase 3: Helix

A `languages.toml` section in `README.md`: a `[[language]]` entry for `tdl` with the comment token, a `[language-server.tdl]` entry running `tdl lsp`, and a `[[grammar]]` entry pointing at this repository with `subpath = "tree-sitter"`.

Helix reads highlight queries from its runtime directory, so the section copies `tree-sitter/queries/highlights.scm` into `runtime/queries/tdl/`.
Capture names that differ from the `nvim-treesitter` names are listed in the section rather than forked into a second query file.
The home-manager module does not configure Helix; the README shows the `programs.helix.languages` equivalent.

Done when `hx --health tdl` reports the server and the grammar, and a conformance file shows a diagnostic and a hover.

## Phase 4: Emacs

`editors/emacs/tdl-mode.el`: a major mode deriving from `prog-mode` with the comment syntax, `auto-mode-alist` for `*.tdl`, and an `eglot-server-programs` entry running `tdl lsp`.
Eglot is built into Emacs 29, so the mode has no dependency.

Font-lock covers keywords, comments, and literals, with a test holding the keyword list to `lex.Keywords()`.
A `tdl-ts-mode` over the tree-sitter grammar through `treesit` can come later beside it.

Done when `emacs -Q` colors a conformance file's keywords, and `eglot` on it shows a diagnostic and a hover.

## Phase 5: Zed

Zed runs a language server only through an extension, written in Rust compiled to WebAssembly: `language_server_command` finds `tdl` on the worktree's `PATH` and returns `tdl lsp`.
It extends the Zed extension from phases 4 and 5 of [editors-plan.md](editors-plan.md).

Done when the extension loaded as a dev extension shows a diagnostic and a hover on a conformance file.

## Phase 6: target-block directives

`tdl gen` checks directives with `gen.CheckDirectives` against each backend's `DirectiveSpec`; nothing checks them while editing.

The server runs the same check for every target block naming a built-in backend, reading `Describe` in process through `gen.Builtin`.
An undeclared directive is a warning, and a wrong argument count or kind is an error, matching `tdl gen`.
A backend on `PATH` is not started, since opening a file should not run an executable.
This is the first non-error diagnostic, so `publish` gains severities.

Done when a misspelled directive in a `target go` block is underlined as it is typed, and the corpus still publishes nothing.

## Phase 7: completion

Completion needs the names in scope at a cursor, and the text being typed usually does not parse.

`internal/sema` gains a way to list the bindings visible from a declaration: the file scope, the prelude, and the declaration's type parameters.
The server keeps the last snapshot that lowered and completes against it, with the lexer deciding the position from the tokens before the cursor.

What a position offers:

- A type position: declarations, type parameters, and names a `_` import merged in; after `alias.`, what that import exports.
- After `:` in a declaration head and after `requires`: classes.
- After `include`: mixins.
- Inside `where { }`: the standard constraint names from `internal/sema/constraint.go`, with their arity.
- Inside a target block: declaration names, then fields and variants after `.`, and the backend's directive names from phase 6 with their argument kinds.

Each item carries its kind and, resolved lazily, the phase 3 hover text.

Done when a field's type offers the model's and the prelude's declarations, `include` offers only mixins, and a directive position in a `target go` block offers `tag` and `key`.

## Phase 8: references, rename, and highlight

These need every occurrence, and the index records every resolution.
`internal/sema` additionally records a declaration's name, a type parameter's, `include`, unit names in a unit expression, constructors in a default, and each segment of a target path (the parser keeps a path as one string, so it has to give per-segment positions).

`textDocument/references` is the records sharing a target, and `textDocument/documentHighlight` is that set within one file.
`textDocument/rename` edits all of them.
`prepareRename` refuses the prelude, a keyword, and a qualified name into a dependency; the new name must be a valid identifier and not reserved, and a rename that collides with a name in scope is refused.

Done when renaming a declaration used in a field, an `include`, a target path, and a file importing it with `_` edits all four, and the corpus still publishes nothing.

## Phase 9: semantic tokens

A name is colored by what it resolved to, so a mixin, a class, an entity, and a type parameter can differ, and a reference to a deprecated declaration carries the `deprecated` modifier.
Token types follow the captures in `tree-sitter/queries/highlights.scm`.
Keywords and literals come from the lexer; names come from the phase 8 index.

Done when a reference to a deprecated declaration is struck through in VS Code and in Neovim.

## Phase 10: structured diagnostics and quick fixes

`sema.Diagnostic` gains a code, a severity, and related locations (such as the first declaration in a duplicate).
Code actions key on codes.
The first fixes: an undefined name offers the closest declared names, and a name another file declares offers the `_` import that brings it in.

Done when an undefined type offers its likely spelling as a quick fix, and applying it clears the diagnostic.

## Not in this plan

- **Publishing** to the Marketplace, `nvim-lspconfig`, MELPA, or the Zed registry.
- **JetBrains.** In [backlog.md](../backlog.md).
- **Folding, selection ranges, workspace symbols, and inlay hints.** Each is additive once completion and the full index exist.
- **The prelude under a `tdl:` URI.** [lsp.md](lsp.md) explains why hover is the answer.
