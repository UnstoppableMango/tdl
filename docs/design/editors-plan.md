# Implementing editor support

An implementation plan for [editors.md](editors.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds highlighting and nothing else.

The language server is described in [workflow.md](workflow.md) and is separate work.
Every target here wants it more than it wants highlighting, and none of these files changes when it arrives.

It touches no part of the reference implementation.
`lex` already states the lexical facts a second grammar needs, `docs/grammar.ebnf` already carries the annotations, and neither is read differently by any of this.

JetBrains and Emacs stay in [backlog.md](../backlog.md).

## Layout

```text
internal/textmate/     # the annotated grammar to tmLanguage.json, private
tools/textmate/        # main, run with go run
editors/vscode/        # package.json, language-configuration.json, syntaxes/
editors/zed/           # extension.toml, languages/tdl/
tree-sitter-tdl        # a repository of its own, generated from tree-sitter/
```

`editors/` holds what an editor installs, the way `tools/` holds what builds it.

Zed's registry entry takes a `path`, so its extension lives here beside the VS Code one.
Zed's `[grammars]` block does not, which is the whole reason the last repository exists.

## Testing

The emitter is Go and gets Go tests.

`tdl.tmLanguage.json` is committed and regenerated, and the diff is read rather than trusted, the same way `tree-sitter/grammar.js` and `ir/ir.pb.go` are.

Each editor is checked by opening a file in it.
Nothing else answers whether the colours arrived, and a plan that pretends otherwise is checking the wrong thing.

`testdata/conformance/constraints/source.tdl` is the file to open: it carries regex literals, ranges, strings, doc comments, and constraint blocks, which is the widest spread in the corpus.

## Phase 1: Neovim

No new artifact, and nothing to build.

`nvim-treesitter`'s `install_info` takes a `location` for a parser in a subdirectory and a `queries` for the query files beside it, which is this repository's shape already.

`README.md` gains a section: `vim.filetype.add` for `*.tdl`, the parser registration against this repository with `location = "tree-sitter"` and `queries = "tree-sitter/queries"`, and `vim.treesitter.language.register`.

First because it depends on nothing and blocks nothing.

Done when pasting the section into a configuration that has never seen TDL highlights a conformance file.

## Phase 2: the TextMate emitter

`internal/textmate` reads what `ebnf.Read` already returns and writes `editors/vscode/syntaxes/tdl.tmLanguage.json`.
`tools/textmate` is the `main` that runs it, beside `tools/treesitter`.

The input is the lexical layer and only the lexical layer.
`lex.Keywords()`, `lex.Punctuation()`, and the six `Pattern` constants are read rather than restated, so a keyword the emitter colours is a keyword the lexer produces.

TextMate is regular expressions over lines, so this is where the approximation lives.
It colours keywords, punctuation, the literals, comments, and doc comments, and it does not colour a declaration name, which needs a parse.

The patterns are Oniguruma rather than Go's `regexp`.
The two agree on everything these six use except escaping, so translating them is the emitter's job and not the caller's.

`scopeName` is `source.tdl`, which is what `tree-sitter/tree-sitter.json` already declares, so both grammars name the language the same thing.

Done when running the emitter twice produces the same bytes, every spelling in `lex.Keywords()` appears in the output, and a keyword added to `lex` without regenerating fails `go test ./internal/textmate`.

## Phase 3: the VS Code extension

`editors/vscode/package.json` contributes the language, the file extension, and the grammar from phase 2.

`language-configuration.json` is hand-written and separate from the grammar: comment markers, brackets, and auto-closing pairs are editor behaviour rather than facts about the syntax, and no generator should guess them.

`flake.nix` gains `packages.vscode-tdl`, built with `pkgs.vscode-utils.buildVscodeExtension` from that directory.
Installing through nix is the whole distribution story for now; the Marketplace is an account, a token, and a release job, and none of it changes a file here.

Done when `nix build .#vscode-tdl` produces an extension that colours a conformance file, and `AGENTS.md` says how to add it to a configuration.

## Phase 4: the grammar repository

`tree-sitter-tdl`, generated from this repository rather than moved out of it.

Moving it out is the obvious reading and the wrong one.
The derivation's value is that a production reaching `docs/grammar.ebnf` and not the derived parser is a diff that fails one CI job, and that job cannot span two repositories.

A release workflow pushes `tree-sitter/`'s contents into the mirror on a tag: `grammar.js`, `src/`, `queries/`, and `tree-sitter.json`, plus the `package.json` a grammar consumer expects and this repository has no use for.
The mirror carries the same tag, so a consumer pinning a revision is pinning a version of the language.

Done when a tag here produces the same tag there, `tree-sitter generate` in a fresh clone of the mirror rewrites nothing, and `tree-sitter parse` in that clone reads a conformance file.

## Phase 5: Zed

`editors/zed/extension.toml` names the grammar by the mirror's repository and a pinned revision.
`languages/tdl/config.toml` carries the file suffix, the comment markers, and the brackets.

`highlights.scm` is copied from `tree-sitter/queries/` when the extension is built rather than duplicated in `editors/zed/`, because Zed reads it from the extension and not from the grammar, and two copies of a query file is the drift this whole design exists to avoid.

Published through `zed-industries/extensions` with `path = "editors/zed"`, which is how that registry spells a monorepo entry.

Done when the extension loaded as a dev extension colours a conformance file, and the registry entry is open as a pull request.

## Not in this plan

- **GitHub.** Linguist wants a TextMate grammar, which phase 2 produces, and at least 2000 files per extension indexed in the last year across many repositories, which is adoption rather than work. Mapping `*.tdl` onto another language in `.gitattributes` would colour files today at the price of every repository claiming to be written in something it is not. GitHub waits for the numbers.
- **Publishing.** The VS Code Marketplace, the Zed registry's review, and npm are distribution questions rather than correctness ones.
- **Every query but `highlights.scm`.** `brackets.scm`, `outline.scm`, `indents.scm`, and injections are each additive once a target reads the first one.
- **The language server**, and the parts of each extension that exist to bundle it.
