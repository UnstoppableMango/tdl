# Implementing editor support

An implementation plan for [editors.md](editors.md).
Phases are ordered by dependency, and each states what makes it done.
Phases 1 to 3 are done.

## Scope

Highlighting only; the language server is [lsp.md](lsp.md).
Nothing in the reference implementation changes: `lex` and the annotations in `docs/grammar.ebnf` already carry what a second grammar needs.

## Layout

```text
internal/textmate/     # the annotated grammar to tmLanguage.json, private
tools/textmate/        # main, run with go run
editors/vscode/        # package.json, language-configuration.json, syntaxes/
editors/zed/           # extension.toml, languages/tdl/
tree-sitter-tdl        # a separate repository, generated from tree-sitter/
```

Zed's registry entry takes a `path`, so its extension lives here; its `[grammars]` block does not, which is why the mirror repository exists.

## Testing

The emitter has Go tests.
`editors/vscode/syntaxes/tdl.tmLanguage.json` is committed and regenerated, and the diff is read like `tree-sitter/grammar.js`'s.

Each editor is checked by opening `testdata/conformance/constraints/source.tdl` in it, the corpus file with the widest spread of regex literals, ranges, strings, doc comments, and constraint blocks.

## Phase 1: Neovim (done)

`README.md` has a section registering the parser against this repository with `location = "tree-sitter"`, for both `nvim-treesitter` branches:

- `main` takes `queries`, registers in a `User TSUpdate` autocommand, and builds through the `tree-sitter` CLI.
- `master` takes `files` naming both C sources, registers through `get_parser_configs()`, and installs no queries for a custom parser, so `highlights.scm` goes on the runtimepath by hand.

Installing a parser colors nothing: on `main`, highlighting is `vim.treesitter.start` from a `FileType` autocommand or `ftplugin/tdl.lua`; on `master`, it is enabled in the plugin's `setup`.
The parser is named for the filetype, so `vim.treesitter.language.register` is not needed, and the section says so.

Done when the section pasted into an empty configuration colors a conformance file with no ERROR node.

## Phase 2: the TextMate emitter (done)

`internal/textmate` reads what `ebnf.Read` returns and writes `editors/vscode/syntaxes/tdl.tmLanguage.json`; `tools/textmate` runs it.
`scopeName` is `source.tdl`, matching `tree-sitter/tree-sitter.json` (`TestScopeNameMatchesTreeSitter`).

`lex.Keywords()`, `lex.Punctuation()`, and the `*Pattern` constants are read, not restated.
The patterns use only syntax Oniguruma and Go's `regexp` spell the same way, so they are copied verbatim, and `TestPatternsComeFromLex` holds the copy to the original.
Modifiers are read from the grammar's quoted terminals; `_` is excluded.

Names colored from a neighboring token:

- A declaration name after the keyword that declares it.
- A field name before `:`.
- A constraint or directive name before `(`.
- A type reference after the punctuation opening a type position, or after `include` and `requires`.
- A target path, found by the `{` or `=>` that follows it and colored whole; the directive after `=>` is colored even without arguments.

Which keyword declares a name and which uses one is read from the production that follows it (`entity identifier` declares, `include NamedType` uses).
The emitter names `NamedType`, `ClassRef`, `TypeRef`, and `EnumDecl`, since the notation does not say what a production means.

Rules that encode language facts:

- A regex literal is matched only after the `(` or `,` opening a constraint argument, so `decimal<kg*m/s^2>` reads as arithmetic.
- Every word rule ends in a `(?!\s*:)` lookahead, since a reserved word followed by `:` is a field name.
- A numeric literal is guarded against the identifier it might sit in.
- A reserved word is never a type reference, so the kinds in `List: type -> type` stay keywords.
- `{` opens a type only with no space after it, separating `{string -> int}` from `where {`.

An enum body is a region: a variant has no neighboring token to identify it, so the block does.
A variant with fields opens a region that includes the whole grammar again, and every `{` opens a region, because a region ends at the first `}` and a nested brace left to the punctuation rule would end it early (`TestEveryOpenBraceIsClosedByItsRegion`).

The class `instance` names stays uncolored: its production puts an optional group between the keyword and the name.

Done when the emitter is deterministic, every `lex.Keywords()` spelling appears in the output, and a keyword added to `lex` and `reserved_word` fails `TestTmLanguage` until `make textmate` runs.

## Phase 3: the VS Code extension (done)

`editors/vscode/package.json` contributes the language, the file extension, and the phase 2 grammar.

`language-configuration.json` is hand-written: comment markers, brackets, and auto-closing pairs are editor behavior.
`<` and `>` are an auto-closing pair and not a bracket pair, because bracket pair colorization would paint the `>` in `->` and `=>` as an unmatched bracket.

`packages.vscode-tdl` builds it with `vscode-utils.buildVscodeExtension`, with `sourceRoot` set because the default assumes a `.vsix` layout.
`package.json`'s version belongs to release-please, through a JSON updater in `release-please-config.json`.
The generated grammar is excluded from treefmt, like `tree-sitter/src/*.json`.

`make vscode-install` packages the directory as a `.vsix` (a zip and two XML files written by `install.sh`, without `vsce`) and installs it with `code --install-extension`.
That is the only route that reaches the client: a directory copied into an extensions folder registers on a remote server and never appears in the Extensions view.

Done when `nix build .#vscode-tdl` produces an extension that colors a conformance file, and `AGENTS.md` says how to add it to a configuration.

## Phase 4: the grammar repository

`tree-sitter-tdl`, generated from this repository rather than moved out of it, so the regeneration check stays one CI job.

A release workflow pushes `tree-sitter/`'s contents into the mirror on a tag: `grammar.js`, `src/`, `queries/`, and `tree-sitter.json`, plus the `package.json` grammar consumers expect.
The mirror carries the same tag.

Done when a tag here produces the same tag there, `tree-sitter generate` in a fresh clone of the mirror rewrites nothing, and `tree-sitter parse` in that clone reads a conformance file.

## Phase 5: Zed

`editors/zed/extension.toml` names the mirror and a pinned revision.
`languages/tdl/config.toml` carries the file suffix, the comment markers, and the brackets.
`highlights.scm` is copied from `tree-sitter/queries/` when the extension is built, so there is one copy in the repository.

Published through `zed-industries/extensions` with `path = "editors/zed"`.

Done when the extension loaded as a dev extension colors a conformance file, and the registry entry is open as a pull request.

## Not in this plan

- **GitHub.** Waits on Linguist's usage threshold; see [editors.md](editors.md).
- **Publishing** to the VS Code Marketplace, the Zed registry, or npm.
- **Query files other than `highlights.scm`.**
- **The language server.** [lsp-editors-plan.md](lsp-editors-plan.md).
