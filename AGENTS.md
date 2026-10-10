# AGENTS.md

Guidance for coding agents working in this repository.

## Commands

```shell
go test ./...                      # all tests
go test -race ./...                # what CI runs; the plugin subprocess needs it
go test -short ./...               # skips TestValidationRuns, which runs go on generated code
go test ./parser -run TestConformanceCorpusParses   # a single test
go build ./...

command make build            # nix build .#
command make test             # go test ./...
command make cover            # go test -coverpkg=./... -coverprofile=cover.profile ./... + go tool cover -func
command make play             # watch examples/nested.tdl; FILE=scratch.tdl VIEWS=all to override
command make demo             # render docs/demo/demo.tape with VHS into docs/demo/demo.gif
command make lint             # nix flake check + golangci-lint + buf + markdownlint
command make check            # nix flake check alone, the fast subset of lint
command make fmt              # nix fmt (treefmt) + buf format
command make update           # nix flake update
command make tidy             # go mod tidy + regenerate nix/gomod2nix.toml
command make generate         # buf generate: proto/ -> ir/ir.pb.go
command make treesitter       # docs/grammar.ebnf -> tree-sitter/grammar.js -> tree-sitter/src/
command make textmate         # docs/grammar.ebnf -> editors/vscode/syntaxes/tdl.tmLanguage.json
command make vscode-install   # package editors/vscode and install it into a running VS Code
command make vscode-check     # npm ci, then typecheck and biome check editors/vscode
command make vscode-test      # in `nix develop .#vscode`: the extension in a headless VSCodium
command make test-treesitter  # the conformance corpus, run by tree-sitter
command make check-treesitter # what CI runs: treesitter + a diff + test-treesitter
```

Prefix `make` with `command` (see the shell autoload note in the global instructions).

`generate`, `treesitter`, and `textmate` are file targets, so each reruns only when its inputs are newer.
`tree-sitter/Makefile` and `editors/vscode/Makefile` hold the targets for their directories; the root delegates to them.
`check-treesitter` passes `-B`, because a fresh checkout gives every file the same mtime and a file target would skip the regeneration the diff is meant to test.
Never combine `-race` with `-coverpkg`: every covered statement becomes an atomic the race detector tracks, and the protocompile tests time out.
CI runs them as two steps.
`vscode-install` regenerates the TextMate grammar first, and `install.sh` refuses to run without a bundle built by `make`.

`nix fmt` formats Go, Nix, YAML, JSON, TOML, Markdown, protobuf, and TypeScript, and `nix flake check` fails on anything unformatted.

`editors/vscode/src/` is the only TypeScript.
`editors/vscode/package-lock.json` pins TypeScript 7 and Biome, run through `npm run`.
`editors/vscode/biome.json` configures Biome as formatter and linter; treefmt reads it, so a lint finding fails `nix flake check`.
Biome's JSON formatter is off, because jsonfmt owns JSON.

`.markdownlint-cli2.yaml` lists which markdown files are linted, so a bare `markdownlint-cli2` checks what CI checks.
Files with no formatter: the three Makefiles, `.editorconfig`, `docs/grammar.ebnf`, `docs/notation.ebnf`, `.github/skills/**/SKILL.md` (mdformat would break the frontmatter), `docs/demo/demo.tape`, `docs/demo/demo.bash`, `tree-sitter/corpus.sh`, `editors/vscode/install.sh`, and `tree-sitter/src/scanner.c`.
`internal/ebnf` lints the two grammars instead, and their column alignment is deliberate.
Also excluded from formatting: `*.tdl` (until `tdl fmt` is wired in, see `docs/backlog.md`), generated files, and `.claude/`.

After changing `go.mod`, run `make tidy` so `nix/gomod2nix.toml` stays in sync; otherwise `nix build` fails.
Renovate does the same for its own updates through the `gomod2nix` preset in `.github/renovate.json`.

Neither `gomod2nix` nor `protoc-gen-go` is on `PATH`.
Run generators through the devShell: `nix develop --command make tidy`, and `buf generate` inside `nix develop` or with `PATH="$(go env GOPATH)/bin:$PATH"`.

## Nix

`nix/` holds the packaging:

- `cmd.nix`: the CLI. `meta.mainProgram` is what `lib.getExe` reads, since the package installs twelve binaries.
- `vscode-extension.nix`: the editor extension (see [VS Code](#vs-code)).
- `demo.nix`: renders `docs/demo/demo.tape`, the README's GIF, with VHS. It is a package and not a check, since it runs a browser.
  The tape's `cat`, from `docs/demo/demo.bash`, highlights TDL with the tree-sitter grammar and the theme in `themes/tree-sitter/`, and everything else with bat.
- `overlay.nix`: names both packages and composes gomod2nix's overlay, so a consumer adding it also gets `buildGoApplication` and `mkGoEnv`.
- `hm-module.nix`: the home-manager module.
- `flake-module.nix`: the flake-parts module a consuming project imports.
- `default.nix`: the flake-parts module `flake.nix` imports; exports the overlay, the modules, and the packages.

`flake.nix` holds the inputs, the devShell, the treefmt configuration, and `version`.
`perSystem` imports nixpkgs with the overlay and reads `packages.default` and `packages.vscode-tdl` back out of it, so `nix build .#` takes the same path a consumer does.

`hm-module.nix` declares `programs.tdl`.
`enable` adds `package` to `home.packages`; `vscode.enable` (default `programs.vscode.enable`) adds `vscode.package` to the profiles in `vscode.profiles` (default `[ "default" ]`).
Both packages default to `pkgs.tdl` and `pkgs.vscode-tdl` through `mkPackageOption`, so the module needs no flake input.
It is exported as `homeModules.default` and `homeManagerModules.default`, each with a `tdl` alias.
`checks.hm-module` evaluates a minimal configuration and asserts both packages land where the options say; it uses `allowUnfree` because `programs.vscode.enable` evaluates the editor.

`flake-module.nix` declares `perSystem.tdl`.
`enable` defines `devShells.tdl` for `inputsFrom`, plus one check per property a model should hold: `tdl-check` parses and lowers, `tdl-fmt` runs `tdl fmt --check`, and `tdl-gen` runs `tdl gen --verify`.
Each check is one invocation over all files.
`files` is a list of strings relative to one `src`, because a `path` is copied into the store alone and an `include` would stop resolving.
`gen.files` is separate and empty by default, because `tdl gen` fails on a file with no target block.
It is exported as `flakeModules.default` with a `tdl` alias.
`checks.flake-module` evaluates a consumer flake against `testdata/conformance/entity` and builds the result, without `tdl-gen`, which needs generated output on disk.

## Architecture

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections.
It has no expressions, control flow, or runtime.
This repository holds the specification and the reference implementation.

The parser reads the whole grammar, and the conformance corpus lowers to `ir` with no diagnostic.
`docs/design/ir-plan.md` phases 1 through 9 are done.
Phase 8b is partial: a dependency's block-scope target directives reach its `ir.Import`, and its declaration-level directives reach the `ir.Extern` naming the declaration; a transitive dependency's do not, and two dependencies are never compared.
The plugin protocol in `docs/design/plugins.md` is complete.

### Front end

- `lex`: hand-written lexer.
  Positions start here and flow through the AST as `ast.Position`.
  Regex literals are scanned only on request via `RescanRegexAt`, because `/` is also unit division.
  `table.go` exports `Keywords`, `Punctuation`, `Lookup`, and the `*Pattern` constants so tools derived from the grammar read what the lexer accepts.
- `parser`: recursive descent producing `*ast.File`.
  Errors accumulate in an `ErrorList`; `syncTop` resynchronizes at the next declaration.
- `ast`: parse tree mirroring the source, names unresolved.
  `ast.Fprint` is the canonical formatting used by `tdl fmt`.
- `internal/ebnf`: linter for `docs/grammar.ebnf` and `docs/notation.ebnf`, built on `golang.org/x/exp/ebnf`.
  It also checks every quoted terminal against `lex`, and reports unterminated comments and strings itself because the library's scanner prints them to stderr.
- `internal/treesitter`: emits `tree-sitter/grammar.js` from the grammar.
  Rule names are snake_case, a hidden production gets a leading underscore, an inline one is substituted into its callers, and rules keep source order.
  `tools/treesitter` runs it.
  `tree-sitter/src/` is committed output of `tree-sitter generate`, except `scanner.c`, which is hand-written and produces `regex_lit` from `valid_symbols` the way the parser calls `lex.RescanRegexAt`.
- `internal/textmate`: emits `editors/vscode/syntaxes/tdl.tmLanguage.json` from the grammar, since VS Code highlighting is TextMate.
  See [TextMate grammar](#textmate-grammar).
  `tools/textmate` runs it.

### Middle

- `internal/sema`: ast to ir.
  Declaration table, interned type and unit tables, sugar lowering, scopes, recursion rules, and the import graph.
  `units.go` runs first, because a unit may be written after one deriving from it and a type argument naming a unit needs its reduction.
  It touches no filesystem: a `Loader` supplies imports, `FSLoader` for real files and `MapLoader` in tests.
  `refs.go` records every name resolution when `WithReferences` is set, for the language server, by hooking the places lowering already resolves names.
  Private.
- `internal/unlower`: ir to ast, the reverse of `sema`, for reverse backends.
  It writes the model's own declarations and leaves to lowering what lowering computes: inherited constraints, mixin fields, struct kinds, and a class directive's expansion.
  `TestCorpusRoundTrips` holds it to lowering the printed corpus back to an equal model.
- `ir`: the resolved model backends consume.
  `ir.pb.go` is generated from `proto/tdl/ir/v1/ir.proto`; `model.go` holds hand-written lookups, and `WithoutPositions` clears positions so `proto.Equal` compares two models.
  Three interned tables, each its own ID space: `Decls`, `Types`, `Units`.
  A unit is interned on its base dimensions, so `decimal<N>` and `decimal<kg*m/s^2>` are one entry in `Types`; `UnitDef` is the declaration and `Unit` what it measures.
  `proto/` and `ir/` are the public compatibility surface.
- `prelude`: the standard prelude, in TDL, embedded with `go:embed`.
  `sema` loads it beneath every file and merges its declarations untagged.
  Lowering knows the sugar's spellings (`List`, `Option`, ...) but not their meaning, so the prelude is replaceable.
- `plugin`: the backend wire protocol, generated from `proto/tdl/plugin/v1/plugin.proto`, plus the framing codec.
  Public.
- `internal/gen`: the compiler side of the plugin protocol: the backend registry, request building, writing returned files, and `Silence`, which drops warnings with an allowed loss code.
  `internal/gen/echo` is a test backend that writes a model as JSON and imports it back; it is not compiled into `tdl`.
- `internal/config`: reads the nearest `tdl.toml` above a file; only its `[lossy]` table so far.
- `internal/cli`: cobra commands (`ast`, `check`, `fmt`, `gen`, `import`, `ir`, `lsp`, `play`, `tokens`, `version`). See [CLI](#cli).
- `internal/lsp`: the language server. See [Language server](#language-server).
- `cmd/tdl`: main.

### Backends

`backend/internal/emit` holds what every generator shares: which declarations belong to the model, reading one target's directives, positioned warnings, case helpers, `Numbers` for field numbering, `Cascade` (skip every declaration naming a skipped one), and `Resolve`, which expands a type reference to the prelude shape or declaration it names.
It also holds the rules more than one target reads: the JSON wire convention (`Discriminant`, `Tag`), `Length` and `Pattern` for the `length` and `matches` constraints, and `PlanInterfaces`, the class plan for a target that writes a class as a nominal interface.
`Resolve` refuses type parameters and arguments, since only Go generates generics.
`backend/internal/irtest` builds `*ir.Model` values for backend tests, seeded with the prelude and the `Entity` class.

Every backend reports what it cannot generate as a positioned warning rather than emitting something wrong, and skips a declaration that names a skipped one, so its output compiles.

- `backend/golang` (`go`): one file per declaration, or one file for the whole target that a `file` directive in the target block names.
  `types.go` maps IR types to Go and walks types itself, since `Resolve` refuses type parameters.
  `generics.go` maps type parameters and infers `comparable` for map keys; `classes.go` makes a class an interface with one unexported marker method; `validate.go` turns `where` constraints into `Validate`; `foreign.go` maps a declaration to another package's type, and an extern to the type its dependency's `go` block generates.
  An enum with no variant fields is a string type with constants; otherwise a sealed interface with a struct per variant.
  `decimal`, `uuid`, and `date` map to placeholders unless a `foreign` directive names a type.
  A `key` directive becomes a `Key()` method, returning a `<Name>Key` struct for several fields.
  It imports: `reverse.go` reads a package with `go/parser`, taking constraints back from `validate`'s messages and keys from `Key`; `annotate.go` writes the loss warnings and, under `roundtrip`, `//tdl:` comment directives, carrying a declaration whole where reading the unannotated files back misses something.
  `Normalize` is its normal form for the round-trip corpus.
  See `docs/design/go-backend.md` and `go-backend-plan.md`.
- `backend/protobuf`: `.proto` files in the directories the package spells, named for its last segment unless a `file` directive says otherwise.
  Proto3 unless an `edition` directive names one; under an edition, `T?` carries no `optional`.
  A fielded enum is a message holding a oneof of per-variant messages; a newtype expands to its base.
  A `oneof` field directive inlines its enum's single-field variants, and an enum used only that way is not emitted.
  `number` pins a number and unpinned members take the lowest free one; `reserved` writes a `reserved` statement, and a pin on a reserved number or name is refused.
  Target-block `import` adds an import to every file; `option` writes a field, value, or statement option.
  A type tagged `service` becomes a service; its fields apply primitives tagged `rpc`, with arguments in a `stream`-tagged primitive streamed.
  Tests compile every response with `bufbuild/protocompile`.
  It imports: `reverse.go` reads files compiled by `compile.go` into an `ast.File` and lowers it, and `annotate.go` writes the loss warnings and, under `roundtrip`, the options `tdl/annotations.proto` declares.
  `Normalize` is its normal form for the round-trip corpus.
- `backend/thrift`: one `.thrift` file per model under `namespace *`, in dependency order.
  A fielded enum is a union of per-variant structs; a newtype is a `typedef`; field ids come from `emit.Numbers`.
  Tests parse and resolve every response with thriftgo.
  It imports one file: `reverse.go` reads thriftgo's AST, which has no positions, so `locate` scans the source for each definition's keyword; `annotate.go` writes the loss warnings and, under `roundtrip`, `tdl.*` annotations, encoding a value so thriftgo's partial unescaping reads it back.
  `Normalize` is its normal form for the round-trip corpus.
- `backend/smithy`: one Smithy IDL 2.0 file per model.
  Each list or map a field holds becomes a named shape (`LineItemList`, `StringLongMap`); a shadowed prelude shape is written `smithy.api#`.
  Non-optional fields are `@required`, and an optional element makes a collection `@sparse`.
  Tests run `smithy validate` when it is on `PATH`; `checks.gen-smithy` always does.
- `backend/graphql`: one schema per model, output types only.
  Primitives GraphQL lacks, including 64-bit and unsigned integers, become custom scalars declared on use.
  A fielded enum is a union of object types; a fieldless variant gets `_: Boolean`; a map is a warning.
  It imports one file: `reverse.go` reads gqlparser's schema document, taking the package from the file's name; `annotate.go` writes the loss warnings and, under `roundtrip`, `@tdl` directives, which the schema then defines.
  Tests validate every response with `vektah/gqlparser`.
- `backend/typescript`: one `.ts` file per model of JSON wire types, no runtime code.
  A set is an array, a map a `Record`, unrepresentable primitives a string, a newtype a plain alias.
  A fielded enum is a union discriminated on `kind`, renamed by a `discriminant` directive.
  `narrow.go` writes a `oneOf`, or an integer `min`/`max` pair spanning at most 16 values, as a literal union, which does not warn.
  Tests run `tsc --noEmit --strict` when it is on `PATH`; `checks.gen-typescript` always does.
- `backend/jsonschema`: one `.schema.json` document per model, every declaration under `$defs`.
  A fielded enum is a `oneOf` discriminated on `kind`, as in TypeScript; a newtype is a definition with its own constraints, and a base newtype is a `$ref`.
  `where` constraints become keywords (`minimum`, `pattern`, `minLength`, ...) chosen by what the constrained type holds, and one with no keyword warns.
  `draft` picks 2020-12 or draft-07, `root` names the declaration the document validates, `id` sets `$id`, and `closed` refuses undeclared properties.
  Tests compile every response with `santhosh-tekuri/jsonschema` and validate instances against it; `checks.gen-jsonschema` runs `check-jsonschema`.
  The definitions come from `backend/internal/jsonschema`, shared with `openapi`; a `Dialect` there states what differs between the documents the two write.
- `backend/openapi`: one OpenAPI document per model holding schemas and an empty `paths`, YAML by default or JSON under `format("json")`.
  `openapi` picks the version: `3.1` (the default, JSON Schema 2020-12), `3.0` (`nullable`, no `$ref` siblings), or `2.0` (`definitions`, `x-nullable`).
  A fielded enum is a `oneOf` of one component per variant with a `discriminator`; 2.0 has no `oneOf`, so there it warns and is skipped.
  `title` and `version` set `info`; `name`, `discriminant`, and `closed` mean what they mean in `jsonschema`.
  Tests validate every response against the OpenAPI Initiative's schema for its version, vendored in `backend/openapi/testdata/`; `checks.gen-openapi` runs `vacuum lint` on all three versions.
- `backend/salesforce`: Salesforce DX source, one file per component.
  An entity is a custom object, with a warning for each field that has no column; values, mixins, and enums are Apex.
  A `key` directive makes a field a unique external ID.
  Tests check the XML is well formed and `checks.gen-salesforce` runs `xmllint`; nothing checks the Apex.
  See `docs/design/salesforce-backend.md`.
- `backend/likec4`: LikeC4 source for an architecture diagram: `tdl.c4`, the specification of element kinds, the same bytes for every model, and one `<package>.c4` holding a package element and one element per structure, enum, newtype, or class; aliases, primitives, and units have none.
  Only phase 1 of `likec4-backend-plan.md` is built: no relationships or views yet.
  No Go library parses LikeC4, so tests compare text, and also run `likec4 validate --no-layout` when `likec4` is on `PATH`.
- `backend/debug`: describes the model it was given, to exercise the protocol.

`cmd/tdl-gen-<name>` serves each backend as a plugin.
`TestHostsAgree` in `internal/gen` holds each to producing the same bytes in process and over a pipe.
A backend that imports implements `plugin.Importer`, declares `Reverse`, and has a `reverse` column in `shipped`, which `TestImportHostsAgree` runs; `TestReverseIsDeclared` keeps the three in step.
A warning a user can silence carries a loss code, written with `emit.Session.Lossy`; `emit.LossCodes` must match the table in `reverse.md`.
A backend added to the registry needs a row in the `shipped` table in `internal/gen/hosts_test.go` (`TestEveryBuiltinHasARow`) and, if shipped, an entry in `nix/cmd.nix` (`TestPackagedBackendsShip`).

`docs/design/schema-backends.md` maps the seven schema backends.
`docs/design/reverse.md` is the import direction, target language to TDL, and `reverse-plan.md` orders it.
`backend/internal/roundtrip` runs a backend forward and back over `testdata/roundtrip/<target>/<case>/`; its `targets` table holds every backend but `debug`, with each one's normal form, and `TestEveryTargetIsCovered` checks it against `cmd/`.
`TestConformanceComesBack` also takes every conformance case without an import through each target that imports.
`testdata/gen/smoke/source.tdl` exercises the whole mapping, with a target block for each schema backend and for `salesforce`; the nix checks generate from it and run each language's tool on the output.

### Tests and goldens

After any change to lowering or `ir.Dump`, run `go test ./internal/sema -update` and read the diff.
`testdata/plugin/` holds recorded protocol exchanges for other implementations to replay, regenerated with `go test ./internal/gen -record`.
`internal/sema/corpus_test.go` asserts the conformance corpus lowers with no diagnostic.

### Docs

`docs/design/` holds designs and plans; each `*-plan.md` names what its phases add, and its opening lines say which are done.
Read those before assuming a design is built: the C#, Haskell, Java, ML-family, and profiles designs have no code yet.
A design describes the target, not the implementation.
`workflow.md` is furthest ahead: its `tdl.toml` project model is unbuilt.
`docs/backlog.md` is wanted, unscheduled work.

## CLI

The root command silences cobra's error printing; a command returns its error and `cmd/tdl` prints it.

`file.go` is shared by every command that reads files: `loadFile` reads and parses one, and `eachFile` walks the arguments, reporting each failure and continuing.
It prints the error alone, since diagnostics and `os.PathError` already name the file.
With more than one file, output is separated by a `==> path <==` banner; `gen` prints only the paths it wrote.
`play` takes one file, and `gen --watch` rejects more than one.
`fmt --check` lists non-canonical files and writes nothing; `fmt -w` keeps each file's mode.

A file named `-` is standard input, shown as `<stdin>` in positions.
`fmt -w` rejects it, and so does `gen`, because imports resolve relative to the importing file and stdin has no directory.
`ir` and `import` accept it.

`import --from <target>` asks a backend for a model, prints it through `internal/unlower`, and lowers the printed source again, writing nothing when that fails.
`gen` and `import` both drop warnings whose loss code `tdl.toml` or `--allow-lossy` allows, and print a code after the message.

`examples/` holds files to experiment with and is outside the conformance corpus.

## Language server

`internal/lsp` serves `tdl lsp` over `go.lsp.dev/protocol`.
It reuses the front end: `parser.Parse` and `sema.Lower` already report every error in one pass.
An overlay `sema.Loader` serves unsaved text of open documents so imports resolve against the editor's state.

- Text synchronization is full.
- `position.go` converts between `lex.Position` byte columns and the protocol's UTF-16 code units.
- `Server` embeds `protocol.UnimplementedServer`, so a feature is one method.
- A file that does not parse publishes only syntax errors; lowering a broken tree would report spurious undefined names.
- Go to definition reads the `sema.WithReferences` index.
  A name from a `_` import jumps into that file, the one case the server reads a file it has not opened.
- Hover prints the declaration with `ast.PrintDecl`, then its deprecation and doc comment; prelude names hover from the embedded source.
- Formatting returns `ast.Fprint` as one edit and refuses a file that does not parse.
- The outline reads the tree, so a broken file still outlines what was recovered.

See `docs/design/lsp.md` and `lsp-plan.md`.

## Specification and conformance

`docs/spec.md` is canonical and `docs/grammar.ebnf` is the formal grammar.
Update both with any grammar or lexer change.

The grammar uses Wirth syntax notation, the Go and Oberon reports' dialect, not ISO 14977: productions end with `.`, sequence items are juxtaposed, and comments are `/* */` and `//`.
`docs/notation.ebnf` defines the notation in itself.
VS Code uses `igochkov.vscode-ebnf` for highlighting only, since ISO tools reject the files.

A lexical name the lexer owns is a production with no expression and a `/*@ token ... */` annotation naming its `lex` symbol.
`reserved_word` is spelled out and checked against `lex.Keywords`.
`ebnf.Read` returns the annotations with the grammar and checks them: a production with no expression needs a `token` binding, and every name an annotation mentions must exist.
`docs/design/treesitter.md` defines the annotations.
`TestDocsAreClean` fails when either file stops linting clean.

### Derived grammars

Both derived grammars are committed and checked by tests that compare, never write: `go test ./internal/treesitter` and `go test ./internal/textmate`.
After a grammar change, run `make treesitter` and `make textmate` and read the diffs.
A keyword added to `lex` and `reserved_word` shows up in both.

`make test-treesitter` runs `tree-sitter/corpus.sh`: `testdata/conformance/*/source.tdl` must parse with no ERROR node and `testdata/invalid/*/source.tdl` must produce one.
It also compiles the hand-written `tree-sitter/queries/highlights.scm`; `TestHighlightsCoverKeywords` checks it covers every keyword.
CI runs `make check-treesitter` in `nix develop .#treesitter`, since the regeneration diff is only stable against the pinned CLI.
That shell holds only go, gnumake, node, and tree-sitter to keep its closure small; node runs `grammar.js`.

### TextMate grammar

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

### VS Code

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

### Corpora

`testdata/conformance/*/source.tdl` must parse and lower to the sibling `ir.golden`.
`testdata/invalid/*/source.tdl` must fail with an error containing the sibling `error.golden`.
Both are plain text so other implementations can run them, and `parser/conformance_test.go` walks them, so adding a directory adds a case.
A case holding a `pending` file is skipped with its text as the reason; the phase implementing the construct deletes it.

Every `.tdl` file in `testdata/conformance/`, `testdata/gen/`, `prelude/`, and `examples/` is canonical: `tdl fmt <file>` prints it back byte for byte (`TestCorpusIsCanonical`).
`tdl fmt` is idempotent.
At the top level, a blank line between comment groups, or between a comment and the following declaration, survives formatting.
Inside a body, the formatter owns blank lines.

### Protobuf

The protos are Editions 2024.
Each file sets `features.field_presence = IMPLICIT` and `features.(pb.go).api_level = API_OPEN`; a field needing presence sets it itself, as `Range.low` does.
`go_package` lives in `buf.gen.yaml` under managed mode.

After changing `proto/`, run `make generate` and commit the `.pb.go` files.
Field numbers are a compatibility promise to plugins: add fields, never renumber or reuse.
`.github/workflows/buf.yml` runs `buf lint`, `buf format`, and `buf breaking` against the base.
It is advisory: it is outside the `required` job, so a reviewer must read it.
A pull request that must break the schema carries the `buf skip breaking` label; toggling it reruns the check.

## Conventions

Whitespace is insignificant and there are no separators: an item ends where the next begins.
Commas are required inside `<...>`, conformance lists, and list literals, and not permitted inside `{ }`.

`where` introduces a constraint block; `requires` introduces class constraints on parameters.

Declaration keywords are reserved.
Modifiers and constraint names (`owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`) are contextual and usable as field names.
A reserved word or contextual modifier followed by `:` is a field name: `include Foo` is an include, `include: Foo` a field.

Identity is conformance to the prelude's `Entity` class, known to the compiler by name.
`ir.StructKind` is computed from that conformance, and the recursion rules read it.
Which fields identify an entity is a target directive.

A `<...>` argument is a type or a unit.
A bare name is recorded as a type reference and resolved by kind; only an operator (`*`, `/`, `^`) or parentheses makes it a unit.

In a target block, directive names and path segments may be reserved words.
A package path segment may be a reserved word after `package` or a target block's `for`, so `package google.type` works.
`Name` in the grammar is that production; every other name is `identifier`.

Directive and constraint arguments are parenthesized and comma separated, since the parser knows no name's arity.

The parser calls `lex.RescanRegexAt` when it wants a regex; nothing else in the lexer takes context.

Comments survive formatting.
A `///` doc comment is a token attached to the next declaration, in `DeclHead.Doc` with each line's position in `DeclHead.DocP`.
A `//` comment is collected on the side into `ast.File.Comments` in source order.
`ast.Fprint` places each by position, on its own line or at the end of the line it was on; a block holding one does not collapse to a line.
Doc and ordinary comments are merged by offset, so they keep their order.

## Review

Copilot code review reads this file.
`.github/skills/code-review/` holds the review procedure and per-area invariants; `do-not-review.md` lists what CI enforces.
`.github/copilot-instructions.md` is always loaded and stays short.
Its `@../AGENTS.md` is expanded by Copilot CLI; skill files expand no references and stand alone.

CodeRabbit reviews only the tip of a stack.
A lower pull request with no comments may be unreviewed; check the CodeRabbit check, not the thread count.

DeepSource (`.deepsource.toml`) runs the `go`, `shell`, `secrets`, and `test-coverage` analyzers.
Coverage comes from the `check` job's `cover.profile`, authenticated with OIDC; the CLI comes from the devShell.
Check a Go style finding against `.golangci.yml` before acting on it.

`main` requires every review thread resolved and no approval.
Reply with what changed, or why nothing did, then resolve.

The one required check is the `required` job in `.github/workflows/ci.yml`.
A new CI job goes in its `needs`, not in the ruleset (declared in Pulumi in `UnstoppableMango/vcs`).

Pull requests are stacked, and GitHub rebases the rest of the stack on merge; reset local branches from the remote afterwards.
`gh pr merge` and `PUT /pulls/{n}/merge` refuse a stacked pull request.
Use `PUT /pulls/{n}/merge-async`, which returns `{"status":"pending"}` and merges a few seconds later.

## Releases

`.github/workflows/release-please.yml` calls the reusable workflow in `unmango/actions` as the thecluster[bot] GitHub App (`vars.RELEASE_APP_CLIENT_ID`, `secrets.RELEASE_APP_PRIVATE_KEY`), so release PRs trigger CI and commits are signed.

release-please owns the version.
Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, `version` in `editors/vscode/package.json` and `package-lock.json`, or `CHANGELOG.md`.
The first two carry an `x-release-please-version` annotation; the JSON files are listed in `release-please-config.json`.

Two versions are not release-please's:

- `metadata.version` in `tree-sitter/tree-sitter.json`: `tree-sitter generate` copies it into `parser.c`, so a release bump would fail CI's regeneration diff. Bump it by hand with a regeneration.
- `specVersion`: tracks `docs/spec.md`. Never give it the annotation.

`CHANGELOG.md` is excluded from treefmt and markdownlint.

The warning that `version.txt` does not exist is harmless.
