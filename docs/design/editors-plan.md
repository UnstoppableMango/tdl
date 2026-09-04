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

`editors/vscode/syntaxes/tdl.tmLanguage.json` is committed and regenerated, and the diff is read rather than trusted, the same way `tree-sitter/grammar.js` and `ir/ir.pb.go` are.

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

Done.
The done-condition was proved rather than assumed: a keyword added to `lex` and to `reserved_word` fails `TestTmLanguage` until `make textmate` runs.

The patterns needed no translating after all.
The six shapes use only what Oniguruma and Go's `regexp` spell the same way, so they are copied verbatim and `TestPatternsComeFromLex` holds the copy to the original.

Three rules are more than a spelling, and each is a fact about the language rather than a preference.
A regex literal is only ever a `ConstraintArg`, so it is matched after the `(` or `,` that introduces one, and no unit expression is coloured as one: `decimal<kg*m/s^2>` reads as arithmetic and `matches(/^[a-z-]+$/)` as a pattern.
A reserved word followed by `:` is a field name, so every word rule ends in a lookahead and `where: string` is a field where `where {` is a block.
A numeric literal is guarded against the identifier it might sit inside, since `\b` does not separate `x` from `1`.

The modifiers are read from the grammar's quoted terminals rather than listed, so `key`, `owned`, and `deprecated` are coloured and a fourth would be too.
`_` is identifier-shaped and is excluded: `import "x" as _` is a name the source declines to give.

Four kinds of name are coloured that the design document said would not be, and none of them needs a parse after all.
A declaration name is spelled after the keyword that declares it, a field name before a `:`, a constraint or directive name before a `(`, and a type reference after the punctuation opening a type position or after `include` and `requires`.

Both keyword sets are read from the productions rather than listed, and the production after the keyword is what sorts them: `entity identifier` declares a name and `include NamedType` uses one.
`NamedType`, `ClassRef`, and `TypeRef` are named in the emitter, since what a production means is the one thing the notation does not say, and reachability would call each of them a dotted identifier.

Two guards keep the type rules honest.
A reserved word is never a type reference, so the kinds in `List: type -> type` stay keywords, and `{` introduces a type only with no space after it, which is what separates `{string -> int}` from the `where {` canonical form writes with one.

An enum variant is the fifth, and it is the one that needed a region rather than a line rule.
Every other name is read from the token beside it; a variant has none, so what says it is a name is the block it sits in.
A variant carrying fields opens a region of its own that includes the whole grammar again, which is how `Some { value: T }` reads as a variant and a field.
`EnumDecl` is named in the emitter the way the type-valued productions are, and the keyword comes from that production rather than from a literal.

A target path is the sixth, found by what follows it rather than what precedes it: an entry is a path and then a block or a `=>`, and nothing else in the language puts a name before either.
It is coloured whole, the way `highlights.scm` captures every segment of one, and the directive after the `=>` is coloured even when it takes no arguments and the constraint rule cannot see it.

What is left uncoloured is the class `instance` names, whose production puts an optional group between the keyword and the name.

## Phase 3: the VS Code extension

`editors/vscode/package.json` contributes the language, the file extension, and the grammar from phase 2.

`language-configuration.json` is hand-written and separate from the grammar: comment markers, brackets, and auto-closing pairs are editor behaviour rather than facts about the syntax, and no generator should guess them.

`<` and `>` are an auto-closing pair and not a bracket pair.
Bracket pair colourisation reads `brackets`, so listing them there paints the `>` in `->` and `=>` as an unmatched closing bracket, in red, over whatever colour the grammar gave it.
Type arguments lose bracket matching, which is the cheaper half of that trade.

`flake.nix` gains `packages.vscode-tdl`, built with `pkgs.vscode-utils.buildVscodeExtension` from that directory.
Installing through nix is the whole distribution story for now; the Marketplace is an account, a token, and a release job, and none of it changes a file here.

Done when `nix build .#vscode-tdl` produces an extension that colours a conformance file, and `AGENTS.md` says how to add it to a configuration.

Done.
`sourceRoot` is set: `buildVscodeExtension` defaults it to the layout inside a `.vsix`, and the source here is a directory in the tree.

`editors/vscode/package.json` carries a version, which makes it release-please's rather than anyone's to edit.
It is listed in `release-please-config.json` as a JSON updater, since JSON cannot carry the `x-release-please-version` comment `flake.nix` and `internal/cli/version.go` use.

The generated grammar is excluded from treefmt, the way `tree-sitter/src/*.json` is: `jsonfmt` and the emitter would otherwise each rewrite it.

`editors/vscode/install.sh` and `make vscode-install` package the directory as a `.vsix` and install it with `code --install-extension`.
That is the only route that reaches the client: a directory copied into an extensions folder registers on a remote server, appears in its `extensions.json`, and never shows up in the Extensions view, which reads as the grammar silently not working.
The `.vsix` is a zip and two XML files, written by the script rather than by `vsce`, which would pull npm in for that.

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
