# Specification and grammars

`docs/spec.md` is canonical and `docs/grammar.ebnf` is the formal grammar.
Update both with any grammar or lexer change.

The grammar uses Wirth syntax notation, the Go and Oberon reports' dialect, not ISO 14977: productions end with `.`, sequence items are juxtaposed, and comments are `/* */` and `//`.
`docs/notation.ebnf` defines the notation in itself.
VS Code uses `igochkov.vscode-ebnf` for highlighting only, since ISO tools reject the files.

A lexical name the lexer owns is a production with no expression and a `/*@ token ... */` annotation naming its `lex` symbol.
`ebnf.Read` returns the annotations with the grammar and checks them: a production with no expression needs a `token` binding, and every name an annotation mentions must exist.
`docs/architecture/treesitter.md` defines the annotations.
`TestDocsAreClean` fails when either file stops linting clean.

## Derived grammars

Both derived grammars are committed and checked by tests that compare, never write: `go test ./internal/treesitter` and `go test ./internal/textmate`.
After a grammar change, run `make treesitter` and `make textmate` and read the diffs.
A keyword added to `lex` and `reserved_word` shows up in both.

`make test-treesitter` runs `tree-sitter/corpus.sh`: `testdata/conformance/*/source.tdl` must parse with no ERROR node and `testdata/invalid/*/source.tdl` must produce one.
It also compiles the hand-written `tree-sitter/queries/highlights.scm`; `TestHighlightsCoverKeywords` checks it covers every keyword.
CI runs `make check-treesitter` in `nix develop .#treesitter`, since the regeneration diff is only stable against the pinned CLI.
That shell holds only go, gnumake, node, and tree-sitter to keep its closure small; node runs `grammar.js`.

## TextMate grammar

TextMate is regular expressions over lines, so `internal/textmate` colors the lexical layer plus names a neighboring token identifies:

- a declaration name, after the keyword that declares it;
- a field name, before `:`;
- a constraint or directive name, before `(`;
- a type reference, after `:`, `<`, `,`, `[`, `{`, `->`, `=`, `include`, or `requires`.

Keyword sets are read from the productions: a keyword followed by `identifier` declares a name, and one followed by `NamedType` or `ClassRef` uses one.
A reserved word is never colored as a type reference.

An enum body is a region, and a variant with fields opens a nested region that includes the whole grammar.
Every `{` opens a region, because a region ends at the first `}` it sees; `TestEveryOpenBraceIsClosedByItsRegion` checks this.
A target entry is a path followed by `{` or `=>`.
The class name after `instance` stays uncolored.

Three rules depend on position: a regex literal matches only after a constraint argument's `(` or `,`; a word rule ends in `(?!\s*:)` so a reserved word before `:` is a field name; a numeric literal is guarded against a surrounding identifier.
TextMate takes the earliest match and then the first rule listed, so rule order settles ties.

The sublime-syntax (`sublime.go`) shares those semantics, so it is a translation of the same rules: a region is a pushed context whose first rule is its end, and `$self` is `main`.
A sublime-syntax capture cannot hold patterns, so the enum header, the one capture that does, carries a `split`: a head that pushes a header context, and the `{` that sets the body in its place.
`checks.bat-tdl` builds bat's cache with it and highlights the conformance corpus, which compiles every regex.

## VS Code

`editors/vscode/` is the extension.
`src/extension.ts` starts `tdl lsp` through `vscode-languageclient`, running `tdl.server.path` or `tdl` from `PATH`.
It looks the executable up itself, so a missing server is one message rather than a retry loop, and the grammar still colors the file.
esbuild bundles it into `dist/extension.js`.

`nix/vscode-extension.nix` builds it with `buildNpmPackage` and `importNpmLock`, so a lock file update needs no hash change.
It rewrites `tdl.server.path`'s default to `lib.getExe tdl` with `jq`, and a `jq -e` over the source fails the build if the setting is renamed.
`checks.vscode-tdl` builds it, typecheck included.
Install it with `programs.tdl.vscode.enable`, through `vscode-with-extensions` or `programs.vscode.profiles.<name>.extensions`, or with `make vscode-install` while iterating on colors.
A copy placed in an extensions folder registers on a remote server and does nothing locally.
After regenerating the grammar, reinstall and reload the window.

`language-configuration.json` is hand-written.
`<` and `>` are an auto-closing pair but not a bracket pair: as brackets, the `>` in `->` and `=>` would be colored as unmatched.
