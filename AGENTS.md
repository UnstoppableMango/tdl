# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## Commands

```shell
go test ./...                      # all tests
go test -race ./...                # what CI runs; the plugin subprocess needs it
go test ./parser -run TestConformanceCorpusParses   # a single test
go build ./...

command make build            # nix build .#
command make test             # go test ./...
command make cover            # go test -race -coverprofile=cover.profile ./... + go tool cover -func
command make play             # watch scratch.tdl; FILE=examples/nested.tdl VIEWS=all to override
command make lint             # nix flake check + golangci-lint + buf + markdownlint
command make fmt              # nix fmt (treefmt) + buf format
command make tidy             # go mod tidy + regenerate nix/gomod2nix.toml
command make generate         # buf generate: proto/ -> ir/ir.pb.go
command make treesitter       # docs/grammar.ebnf -> tree-sitter/grammar.js -> tree-sitter/src/
command make textmate         # docs/grammar.ebnf -> editors/vscode/syntaxes/tdl.tmLanguage.json
command make vscode-install   # package editors/vscode and install it into a running VS Code
command make test-treesitter  # the conformance corpus, run by tree-sitter
```

Prefix `make` with `command` (see the shell autoload note in the global instructions).

`nix fmt` formats Go, Nix, YAML, JSON, TOML, Markdown, and protobuf; `nix flake check` fails when anything is unformatted.

Which markdown files are linted lives in `.markdownlint-cli2.yaml`, so a bare `markdownlint-cli2` locally checks what CI checks.
`CLAUDE.md` is ignored: its whole content is an import pointing at this file, and a file that is one directive has no heading to lint.
`.github/copilot-instructions.md` and `.github/skills/` are prose and are linted, because they say things this file does not.
Eight files have no formatter: `Makefile`, `.editorconfig`, `docs/grammar.ebnf`, `docs/notation.ebnf`, `.github/skills/**/SKILL.md`, `tree-sitter/corpus.sh`, `editors/vscode/install.sh`, and `tree-sitter/src/scanner.c`.
The two grammars have no published formatter, and their column alignment is chosen per section for reading; `internal/ebnf` lints them instead.
A skill's YAML frontmatter is how Copilot decides when to load it, and mdformat rewrites it into a thematic break.
Deliberately excluded: `*.tdl` (until `tdl fmt` is wired in, see `docs/backlog.md`), `*.golden` and `nix/gomod2nix.toml` and `flake.lock` and `tree-sitter/src/*.json` and `editors/vscode/syntaxes/*.json` (generated), and `.claude/` (agent skills).

After changing `go.mod` or adding dependencies, run `make tidy` so `nix/gomod2nix.toml` stays in sync, otherwise `nix build` fails.

`nix/` holds the packaging: `cmd.nix` is the CLI, `vscode-extension.nix` is the editor extension, `overlay.nix` names both, `hm-module.nix` is the home-manager module, `flake-module.nix` is the flake-parts module a consumer imports, and `default.nix` is the flake-parts module `flake.nix` imports, exporting that overlay as `flake.overlays.default` and holding the packages.
`flake.nix` keeps the inputs, the devShell, the treefmt configuration, and `version`, which release-please rewrites there and the module reads as a module argument.
The module's `perSystem` imports nixpkgs with the overlay and reads `packages.default` and `packages.vscode-tdl` back out of it, so `nix build .#` takes the path a consumer takes rather than a second one beside it.
The overlay composes gomod2nix's, because `tdl` is built with `buildGoApplication`: a consumer adds one overlay and gets `buildGoApplication` and `mkGoEnv` along with `tdl`.

`hm-module.nix` declares `programs.tdl`: `enable` puts `package` in `home.packages`, and `vscode.enable` puts `vscode.package` in the profiles `vscode.profiles` names, defaulting to `programs.vscode.enable` and to `[ "default" ]`.
Both packages default to `pkgs.tdl` and `pkgs.vscode-tdl` through `mkPackageOption`, so the module names no flake input and the overlay stays the one thing a configuration adds.
`default.nix` exports it as `flake.homeModules.default` and, under the name most configurations already reference, as `flake.homeManagerModules.default`; `tdl` is an alias of each.
`checks.hm-module` is what holds it to that: it evaluates a minimal `homeManagerConfiguration` and asserts the two packages landed where the options promise.
It imports its own nixpkgs with `allowUnfree`, because `programs.vscode.enable` evaluates the editor, and it reads the evaluated options rather than `activationPackage`, which would build the editor to say the same thing.

`flake-module.nix` is the other direction: `hm-module.nix` installs the language for a person, and this one wires it into a project that uses it.
It declares `tdl` under `perSystem`, where `enable` defines `devShells.tdl` for a consumer to pull in with `inputsFrom`, and the checks are one per property a model should hold: `tdl-check` parses, `tdl-fmt` diffs each file against `tdl fmt`, and `tdl-gen` runs `tdl gen --verify`.
`files` is relative strings over one `src` rather than a `listOf path`, because a path is copied into the store on its own and an `include` resolves through `sema.FSLoader` relative to the file that writes it, so a split model would stop resolving.
`gen.files` is a separate list and empty by default, because `tdl gen` fails on a file declaring no target block and so it cannot default to `files`.
`fmt.enable` is a switch rather than the check being unconditional, because `tdl fmt` drops ordinary `//` comments and a file carrying them can never pass.
`package` is `pkgs.tdl` through `mkPackageOption` for the same reason it is in `hm-module.nix`.
`default.nix` exports it as `flake.flakeModules.default`, with `tdl` as an alias, and `checks.flake-module` holds it to that: it evaluates a consumer flake against `testdata/conformance/entity` and builds what came out.
The fixture is a conformance case because the corpus is already held to both properties asserted there, parsing clean and being stored in canonical form; `tdl-gen` is left out of it, since `--verify` compares against generated output on disk and no fixture here has any.

Neither `gomod2nix` nor `protoc-gen-go` is on `PATH`.
Run generators through the devShell: `nix develop --command make tidy`, or `gomod2nix --dir . --outdir nix` directly, and `buf generate` either inside `nix develop` or with `PATH="$(go env GOPATH)/bin:$PATH"`.

## Architecture

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections.
No expressions, no control flow, no runtime.
This repo owns both the specification and the reference implementation.

The parser reads the whole grammar, and lowering to `ir` reads all of it: `docs/design/ir-plan.md` phases 1 through 9 are done, and the conformance corpus lowers with no diagnostic.
What is left is phase 8b, merging a dependency's target blocks, which the plan gates on something needing a dependency's blocks.
The plugin protocol in `docs/design/plugins.md` is complete.

Pipeline, one package per stage:

- `lex` — hand-written lexer.
  `lex.Kind` covers idents, literals, keywords, and punctuation; `LookupIdent` turns an identifier into a keyword kind.
  Positions originate here and flow through the AST as `ast.Position` (a type alias).
  Regex literals are scanned only on request via `RescanRegexAt`, because `/` is also division in a unit expression.
  `table.go` states the same lexical facts for a program rather than a person: `Keywords`, `Punctuation`, `Lookup`, `Spelling`, and `Pattern`, so a tool deriving a second parser from `docs/grammar.ebnf` reads what the lexer accepts instead of restating it.
- `internal/ebnf` — the linter for `docs/grammar.ebnf` and `docs/notation.ebnf`.
  Parsing and reachability come from `golang.org/x/exp/ebnf`, which documents this exact dialect; what is local is the check no library makes, holding every quoted terminal to `lex` so a spelling the grammar invents is an error rather than a rule that can never match.
  It also reports an unterminated comment or string itself, because the library hands its scanner no error handler and `text/scanner` prints those to stderr.
  `Options` carries the start symbol; `GrammarOptions` and `NotationOptions` are the two files.
  Private.
- `internal/treesitter` — the emitter: what `internal/ebnf` read, written as `tree-sitter/grammar.js`.
  It knows the notation not at all and tree-sitter entirely, which is the seam between the two packages.
  Rule names are snake_case, a hidden production gains a leading underscore, an inline one is substituted into its callers, and rules are emitted in the order the productions are written.
  Private.
  `tools/treesitter` is the `main` that runs it, a build tool rather than something shipped, which is why it is not under `cmd/`.
  `tree-sitter/src/` is what `tree-sitter generate` writes from `grammar.js` and is committed like `ir/ir.pb.go`; `tree-sitter/src/scanner.c` is hand-written and lives beside it: it produces `regex_lit`, the one token the generated lexer cannot, reading tree-sitter's `valid_symbols` the way the reference parser calls `lex.RescanRegexAt`.
- `internal/textmate` — the second emitter, reading the same file and writing `editors/vscode/syntaxes/tdl.tmLanguage.json`, because VS Code's highlighting is TextMate and cannot read a tree-sitter grammar.
  TextMate is regular expressions over lines, so what it writes is the lexical layer plus the names a neighboring token gives away: a declaration name, spelled after the keyword that declares it; a field name, spelled before a `:`; a constraint or directive name, spelled before a `(`; and a type reference, spelled after the punctuation that opens a type position (`:`, `<`, `,`, `[`, `{`, `->`, `=`) or after `include` and `requires`.
  Both keyword sets are read from the productions rather than listed, and which set a keyword lands in is which production follows it: an `identifier` is a name being declared, and a `NamedType` or `ClassRef` is one being used.
  A reserved word is never colored as a type reference, which is what keeps the kinds in `List: type -> type` reading as keywords.
  An enum body is a region rather than a line rule: a variant has no token beside it to read, so what makes it a name is the block, and a variant carrying fields opens a region of its own that includes the whole grammar again.
  Every `{` opens a region for the same reason, since a region ends at the first `}` it sees and a brace left to the punctuation rule inside one would end it early: a set type or a constraint block inside a variant would uncolor everything after it.
  `TestEveryOpenBraceIsClosedByItsRegion` is that invariant.
  The keyword opening it is read from `EnumDecl`.
  A target entry is a path and then a `{` or a `=>`, and both are what find it, so `User.email => tag("json:email")` colors as a path and a directive.
  What stays uncolored is the class named after `instance`, whose production puts an optional group between the keyword and the name.
  Three rules carry what position says and the shapes cannot: a regex literal is matched after the `(` or `,` that introduces a constraint argument, so unit division is never colored as one; a word rule ends in a `(?!\s*:)` lookahead, because a reserved word followed by `:` is a field name; and a numeric literal is guarded against the identifier it might sit in.
  Rule order settles a tie rather than a race, since TextMate takes the earliest match and then the first rule listed.
  Private.
  `tools/textmate` is the `main` that runs it, beside `tools/treesitter`, and `editors/vscode/` is what an editor installs.
- `parser` — recursive descent over the token stream, producing `*ast.File`.
  Errors accumulate in an `ErrorList` rather than aborting: `syncTop` resynchronizes at the next declaration so one bad line does not swallow the rest of the file.
- `ast` — parse tree mirroring source 1:1, names left unresolved.
  `ast.Fprint` produces the canonical formatting used by `tdl fmt`.
- `internal/cli` — cobra commands (`ast`, `check`, `fmt`, `gen`, `ir`, `play`, `tokens`, `version`) wired in `root.go`.
  The root silences cobra's error printing so a diagnostic list renders as itself; `cmd/tdl` prints whatever a command returns, so a command should return an error rather than print it.
  `play` is a watch-mode playground that re-renders a file on save; `examples/` holds files to experiment with and is outside the conformance corpus.
  `file.go` is what every command that reads a file goes through: `loadFile` reads and parses one, and `eachFile` walks the arguments, reporting each failure and continuing rather than stopping at the first, the way the parser reports every syntax error in a file.
  It prints the error and not the path beside it, because a diagnostic and an `os.PathError` both already name the file.
  `writeHeader` is the `==> path <==` banner, written only when there is more than one file, so single-file output stays pipeable.
  `play` is the one command still taking a single file, and `gen --watch` rejects a second one, since neither returns.
- `ir` — the resolved model backends consume.
  `ir.pb.go` is generated from `proto/tdl/ir/v1/ir.proto` by `make generate` and committed; `model.go` holds the hand-written lookups.
  Three interned tables: `Decls`, `Types`, and `Units`, each its own ID space.
  A unit is a quantity reduced to base dimensions and interned on them, so `decimal<N>` and `decimal<kg*m/s^2>` are one entry in `Types`; `UnitDef` is the declaration and `Unit` is what it measures.
  `proto/` and `ir/` are the public compatibility surface.
- `cmd/tdl-gen-debug` — the debug backend as a plugin.
  The same value the registry holds, served over a connection, which is what makes the two hosts testable against each other.
- `plugin` — the wire protocol a backend speaks, generated from `proto/tdl/plugin/v1/plugin.proto`, plus the framing codec.
  Public, like `ir`.
- `internal/gen` — the compiler side of the plugin protocol: which backends exist, how a request is built, and what happens to the files that come back.
  Private.
- `backend/debug` — a backend that describes the model it was given.
  Useless on purpose: it exercises the protocol without anyone agreeing what generated code should look like.
- `internal/sema` — ast to ir: the declaration table, the interned type and unit tables, sugar lowering, scopes, the spec's recursion rules, and the import graph.
  `units.go` runs before the rest of lowering, because a unit may be written after the unit deriving from it and because a type argument naming a unit needs its reduction already computed.
  It touches no filesystem: a `Loader` supplies imported sources, with `FSLoader` for real files and `MapLoader` for tests.
  Private and free to change.
  See `docs/design/ir-plan.md` for what each phase adds.
- `prelude` — the standard prelude, written in TDL and embedded with `go:embed`.
  `sema` loads it into an outer scope beneath every file and merges its declarations into the model untagged.
  Lowering knows the sugar's spellings (`List`, `Option`, ...) but nothing about what they mean, which is what makes the prelude replaceable.
- `cmd/tdl` — main.

There are no code-generation backends.
`docs/design/plugins.md` describes the protocol they will speak.

Regenerate the ir goldens with `go test ./internal/sema -update` after any change to lowering or to `ir.Dump`, and read the diff rather than trusting it.

`testdata/plugin/` holds recorded protocol exchanges as protobuf text, regenerated with `go test ./internal/gen -record`.
They exist for an implementation in another language to replay.

`internal/sema/corpus_test.go` asserts the conformance corpus lowers with no diagnostic at all.
It used to carry a `deferred` list naming the phase that would stop producing each one; units were the last entry, so the list is gone.

## Specification and conformance

`docs/spec.md` is canonical; `docs/grammar.ebnf` holds the formal grammar.
Both must be updated alongside any grammar or lexer change.

The grammar is written in Wirth syntax notation, the dialect the Go and Oberon reports use, not ISO 14977: a production ends with `.` rather than `;`, and items in a sequence are juxtaposed rather than separated by `,`.
`docs/notation.ebnf` describes that notation in itself, and `golang.org/x/exp/ebnf` parses it, which is what `internal/ebnf` reads both files with.
Comments are Go's `/* */` and `//` for that reason, not the ISO family's `(* *)`.
An ISO EBNF tool reads either file as an error from its first production, so VS Code is pointed at `igochkov.vscode-ebnf` for highlighting alone.
A lexical name the lexer owns is declared as a production with no expression, which is how this notation says the name is defined elsewhere, and a `/*@ token ... */` annotation names the `lex` symbol that defines it.
`reserved_word` is spelled out rather than annotated, and `internal/ebnf` checks that list against `lex.Keywords`, so a keyword added to one and not the other fails the build.
`ebnf.Read` returns the annotations beside the grammar, scanned from the comments the parsing library drops, and holds the file to them: a production with no expression needs a `token` binding, and every name an annotation mentions has to exist.
`docs/design/treesitter.md` defines them, and `internal/treesitter` is what reads them: `tree-sitter/grammar.js` is derived from this file with `go run ./tools/treesitter`, committed like `ir/ir.pb.go`, and checked by `go test ./internal/treesitter`, which rewrites it under `-update`.
`make treesitter` runs that and `tree-sitter generate` after it, since the parser is committed too.
Change the grammar and regenerate, and read the diff rather than trusting it.
`make test-treesitter` runs `tree-sitter/corpus.sh`, which holds the derived parser to both corpora: `testdata/conformance/*/source.tdl` must parse with no ERROR node and `testdata/invalid/*/source.tdl` must produce one.
The invalid half checks the ERROR and not the message, since `error.golden` is the reference implementation's wording.
It also compiles `tree-sitter/queries/highlights.scm`, which is hand-written: what should be colored is a judgment rather than a fact about the grammar, so the generator does not emit it.
Compiling it catches a node the grammar no longer has, and `TestHighlightsCoverKeywords` catches a keyword it never gained, since a keyword is an anonymous token no tree carries the name of.
CI runs `make treesitter`, `git diff --exit-code`, and `make test-treesitter` through the devShell, because the regeneration diff is only stable against the CLI version `flake.lock` pins.
`TestDocsAreClean` fails the build when either file stops linting clean.

The same file derives the VS Code grammar.
`editors/vscode/syntaxes/tdl.tmLanguage.json` is written by `go run ./tools/textmate`, committed, and checked by `go test ./internal/textmate`, which rewrites it under `-update`; `make textmate` is that command.
A keyword added to `lex` and to `reserved_word` is a diff there as well as in `grammar.js`, so both grammars move together.
The extension around it is `editors/vscode/`, and `nix build .#vscode-tdl` builds it; `programs.tdl.vscode.enable` in the home-manager module installs it, and by hand it goes in `vscode-with-extensions` or in home-manager's `programs.vscode.profiles.<name>.extensions`, either from `packages.vscode-tdl` or as `pkgs.vscode-tdl` through the overlay.
`make vscode-install` is the other way in, for iterating on the colors: it packages the directory as a `.vsix` and hands it to `code --install-extension`, which is the only route that reaches the client.
A directory copied into an extensions folder registers on a remote server and stops there, which looks exactly like the grammar not working.
The installed copy is a copy, so a regenerated grammar needs the command again, and the window needs a reload.
`language-configuration.json` is hand-written on purpose: comment markers, brackets, and auto-closing pairs are editor behavior rather than facts about the syntax.
`<` and `>` are an auto-closing pair and deliberately not a bracket pair: bracket pair colorization reads `brackets`, and listing them there paints the `>` in `->` and `=>` red as an unmatched closing bracket, over the color the grammar gave it.

`testdata/conformance/*/source.tdl` must parse cleanly and lower to the tree in the sibling `ir.golden`; `testdata/invalid/*/source.tdl` must fail with an error containing the text in the sibling `error.golden`.
Both corpora are plain text, deliberately not Go code, so a non-Go implementation can run the same checks.
`parser/conformance_test.go` walks them automatically, so adding a directory is enough to add a case.

A case directory holding a `pending` file describes a construct the parser cannot read yet and is skipped, with the file's text as the skip reason.
The phase that implements the construct deletes the marker.
The corpus is the written-down target, not a record of what already works.

`tdl fmt` must be idempotent: formatting canonical output is a no-op.

Every `.tdl` file in `testdata/conformance/` and `prelude/` is stored in canonical form: `tdl fmt <file>` must print it back byte for byte.
`examples/*.tdl` deliberately are not, because they carry `//` comments.

The protos are Protobuf Editions 2024.
Editions default every field to explicit presence and the generated Go to the opaque API, so each file sets `features.field_presence = IMPLICIT` (what proto3 meant) and `features.(pb.go).api_level = API_OPEN` (the API these types were published with).
A field wanting presence says so itself, as `Range.low` does.
`go_package` lives in `buf.gen.yaml` under managed mode, not in the files.

After changing `proto/`, run `make generate` and commit `ir/ir.pb.go` with it.
Field numbers are a compatibility guarantee to plugins: add fields, never renumber or reuse them.
CI enforces that with `buf breaking` against the pull request's base, alongside `buf lint` and `buf format`; `make fmt` formats the protos and `make lint` checks them.

A pull request that has to break the schema carries the `buf skip breaking` label, which is what `bufbuild/buf-action` reads.
The workflow only re-runs on push, so label first and then push, or the run will still be working from a payload without it.

## Copilot

Copilot code review reads this file directly on github.com, so architecture is already covered and is not what the other two places are for.

`.github/skills/code-review/` is an agent skill: `SKILL.md` plus reference files Copilot picks up alongside it, loaded when its description matches the work.
It holds the review procedure and the per-area invariants, and `do-not-review.md` is the part that matters most, because the default failure is comments on things CI already enforces.
Copilot's own review comments recommend this path.

`.github/copilot-instructions.md` is always loaded and stays short: orientation, a pointer at the skill, and the two conventions that look like defects.

The leading `@../AGENTS.md` there is a file reference Copilot CLI expands, and which code review does not need because it reads this file anyway.
References are not expanded inside a skill file, so `SKILL.md` and its siblings stand alone.

Keep all of it short.
It earns its place by saying what a reviewer would otherwise get wrong, not by describing the repository.

## Conventions

Whitespace is insignificant and there are no separator rules: an item ends where the next begins.
Commas are required inside `<...>`, conformance lists, and list literals, and are not permitted inside `{ }` blocks.

`where` introduces a constraint block; `requires` introduces class constraints on parameters.
Both readings of `{` would otherwise collide.

Declaration keywords are reserved.
Modifiers and constraint names (`key`, `owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`) are contextual and remain usable as field names.

A reserved word followed by `:` is a field name: `value: T` is a field, and `include Foo` is still an include while `include: Foo` is a field.
A contextual modifier followed by `:` is likewise a name, not a modifier.

A class may not declare key fields, so `key` inside a class body is always the requirement.
A class says an implementor must have identity, never which field carries it.

A `<...>` argument is a type or a unit.
A bare name could be either, so it is recorded as a type reference and the resolver picks by kind; only an operator (`*`, `/`, `^`) or parentheses makes it unambiguously a unit.

Inside a target block a directive name and a path segment may be reserved words, since that namespace belongs to the backend.
`Name` in `docs/grammar.ebnf` is the production for a name that may be spelled with one; every other name is a plain `identifier`.

Directive and constraint arguments are parenthesized and comma separated.
Both sets are open, so the parser knows no name's arity and an unparenthesized `min 0 max 100` could not be split.

Regex literals are ambiguous with unit division, so the parser calls `lex.RescanRegexAt` when it wants one.
Nothing else in the lexer takes context.

`tdl fmt` drops ordinary `//` comments: the lexer skips them and they never reach the AST.
Doc comments (`///`) survive.
Fixing this needs comment attachment in the parser and has no phase yet.
Never run `tdl fmt` over `examples/*.tdl`: it silently deletes their explanatory headers.

## Releases

release-please owns the version.
Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, `version` in `editors/vscode/package.json`, or `CHANGELOG.md`; each release PR rewrites them.
The first two carry an `x-release-please-version` annotation, which is what makes them update rather than drift.
JSON has no comments, so the extension's is a JSON updater in `release-please-config.json` naming `$.version` instead.

`metadata.version` in `tree-sitter/tree-sitter.json` is not release-please's either, and for a harder reason than `specVersion`'s.
`tree-sitter generate` parses it into `tree-sitter/src/parser.c` as `.metadata.major_version`, `.metadata.minor_version`, and `.metadata.patch_version`, so a bump release-please writes is a bump nothing regenerates, and the next `make treesitter` produces a diff that fails CI on every pull request until someone notices.
It is the grammar's own version, bumped by hand alongside a regeneration; the mirror repository in `docs/design/editors.md` stamps the release version into its copy.

`specVersion` is not release-please's and must not be given the annotation.
It tracks `docs/spec.md`, which moves on its own schedule, and a release that changed no spec text should not claim to have changed the spec.

`CHANGELOG.md` is excluded from treefmt and from markdownlint, because a generated file that a formatter rewrites is a file the next release will fight over.

The manifest starts at `0.0.34`, the last release the legacy implementation made, and the first release is `0.1.0`.
That is computed rather than forced: `feat!: rewrite the lexer and parser` is a breaking change, and `bump-minor-pre-major` turns a breaking change on a `0.x` version into a minor bump.
Nothing carries a `release-as` override, so nothing has to be removed afterwards.

`release-please` warns that `version.txt` does not exist on every run.
The `simple` release type looks for one by default; this repository states its version in the files that read it instead, and the warning is harmless.
