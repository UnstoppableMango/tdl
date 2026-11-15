# Formatting

`nix fmt` formats Go, Nix, YAML, JSON, TOML, Markdown, protobuf, and TypeScript, and `nix flake check` fails on anything unformatted.

`editors/vscode/src/` and `js/` are the only TypeScript.
Each `package-lock.json` pins TypeScript 7 and Biome, run through `npm run`.
`editors/vscode/biome.json` configures Biome as formatter and linter; treefmt reads it, so a lint finding fails `nix flake check`.
treefmt also runs it over `js/**/*.ts`, but for the generated code, and `npm/**/*.js`, the npm package's shim.
`js/biome.json` carries the same formatter settings and refuses Bun-only imports.
Biome's JSON formatter is off, because jsonfmt owns JSON.

`.markdownlint-cli2.yaml` lists which markdown files are linted, so a bare `markdownlint-cli2` checks what CI checks.
Files with no formatter: the four Makefiles, `.editorconfig`, `.gitattributes`, `docs/grammar.ebnf`, `docs/notation.ebnf`, `.github/skills/**/SKILL.md` (mdformat would break the frontmatter), `.graphifyignore`, `docs/demo/demo.tape`, `docs/demo/demo.bash`, `tree-sitter/corpus.sh`, `editors/vscode/install.sh`, `backend/typescript/outline.js`, `tree-sitter/src/scanner.c`, and everything in `dotnet/` but `deps.json`.
rustfmt formats `rust/`, but for `Cargo.lock` and the generated code.
`internal/ebnf` lints the two grammars instead, and their column alignment is deliberate.
Also excluded from formatting: `*.tdl`, generated files, and `.claude/`.
