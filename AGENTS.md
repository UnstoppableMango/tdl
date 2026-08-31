# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## Commands

```shell
go test ./...                      # all tests
go test ./parser -run TestConformanceCorpusParses   # a single test
go build ./...

command make build     # nix build .#
command make test      # go test ./...
command make cover     # go test -coverprofile=cover.profile ./... + go tool cover -func
command make play      # watch scratch.tdl; FILE=examples/nested.tdl VIEWS=all to override
command make lint      # nix flake check + golangci-lint + buf + markdownlint
command make fmt       # nix fmt (treefmt) + buf format
command make tidy      # go mod tidy + regenerate nix/gomod2nix.toml
command make generate  # buf generate: proto/ -> ir/ir.pb.go
```

Prefix `make` with `command` (see the shell autoload note in the global instructions).

`nix fmt` formats Go, Nix, YAML, JSON, TOML, Markdown, and protobuf; `nix flake check` fails when anything is unformatted.

Which markdown files are linted lives in `.markdownlint-cli2.yaml`, so a bare `markdownlint-cli2` locally checks what CI checks. `CLAUDE.md` and `.github/copilot-instructions.md` are ignored: their whole content is an import pointing at this file, and a file that is one directive has no heading to lint. Three files have no formatter: `Makefile`, `.editorconfig`, and `docs/grammar.ebnf`. Deliberately excluded: `*.tdl` (until `tdl fmt` is wired in, see `docs/backlog.md`), `*.golden` and `nix/gomod2nix.toml` and `flake.lock` (generated), and `.claude/` (agent skills).

After changing `go.mod` or adding dependencies, run `make tidy` so `nix/gomod2nix.toml` stays in sync, otherwise `nix build` fails.

Neither `gomod2nix` nor `protoc-gen-go` is on `PATH`. Run generators through the devShell: `nix develop --command gomod2nix --dir . --outdir nix`, and `buf generate` either inside `nix develop` or with `PATH="$(go env GOPATH)/bin:$PATH"`.

## Architecture

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections. No expressions, no control flow, no runtime. This repo owns both the specification and the reference implementation.

The parser reads the whole grammar. Lowering to `ir` has started; `docs/design/ir-plan.md` phases 1 through 8 are done. What is left is phase 8b, merging a dependency's target blocks, and units, which `ir.md` defers. After that comes the plugin protocol in `docs/design/plugins.md`.

`union` is the one grammar form the parser does not implement.

Pipeline, one package per stage:

- `lex` — hand-written lexer. `lex.Kind` covers idents, literals, keywords, and punctuation; `LookupIdent` turns an identifier into a keyword kind. Positions originate here and flow through the AST as `ast.Position` (a type alias). Regex literals are scanned only on request via `RescanRegexAt`, because `/` is also division in a unit expression. `table.go` states the same lexical facts for a program rather than a person: `Keywords`, `Punctuation`, `Lookup`, `Spelling`, and `Pattern`, so a tool deriving a second parser from `docs/grammar.ebnf` reads what the lexer accepts instead of restating it.
- `parser` — recursive descent over the token stream, producing `*ast.File`. Errors accumulate in an `ErrorList` rather than aborting: `syncTop` resynchronizes at the next declaration so one bad line does not swallow the rest of the file.
- `ast` — parse tree mirroring source 1:1, names left unresolved. `ast.Fprint` produces the canonical formatting used by `tdl fmt`.
- `internal/cli` — cobra commands (`ast`, `check`, `fmt`, `gen`, `ir`, `play`, `tokens`, `version`) wired in `root.go`. The root silences cobra's error printing so a diagnostic list renders as itself; `cmd/tdl` prints whatever a command returns, so a command should return an error rather than print it. `play` is a watch-mode playground that re-renders a file on save; `examples/` holds files to experiment with and is outside the conformance corpus.
- `ir` — the resolved model backends consume. `ir.pb.go` is generated from `proto/tdl/ir/v1/ir.proto` by `make generate` and committed; `model.go` holds the hand-written lookups. `proto/` and `ir/` are the public compatibility surface.
- `cmd/tdl-gen-debug` — the debug backend as a plugin. The same value the registry holds, served over a connection, which is what makes the two hosts testable against each other.
- `plugin` — the wire protocol a backend speaks, generated from `proto/tdl/plugin/v1/plugin.proto`, plus the framing codec. Public, like `ir`.
- `internal/gen` — the compiler side of the plugin protocol: which backends exist, how a request is built, and what happens to the files that come back. Private.
- `backend/debug` — a backend that describes the model it was given. Useless on purpose: it exercises the protocol without anyone agreeing what generated code should look like.
- `internal/sema` — ast to ir: the declaration table, the interned type table, sugar lowering, scopes, the spec's recursion rules, and the import graph. It touches no filesystem: a `Loader` supplies imported sources, with `FSLoader` for real files and `MapLoader` for tests. Private and free to change. See `docs/design/ir-plan.md` for what each phase adds.
- `prelude` — the standard prelude, written in TDL and embedded with `go:embed`. `sema` loads it into an outer scope beneath every file and merges its declarations into the model untagged. Lowering knows the sugar's spellings (`List`, `Option`, ...) but nothing about what they mean, which is what makes the prelude replaceable.
- `cmd/tdl` — main.

There are no code-generation backends. `docs/design/plugins.md` describes the protocol they will speak.

Regenerate the ir goldens with `go test ./internal/sema -update` after any change to lowering or to `ir.Dump`, and read the diff rather than trusting it.

`testdata/plugin/` holds recorded protocol exchanges as protobuf text, regenerated with `go test ./internal/gen -record`. They exist for an implementation in another language to replay.

`internal/sema/corpus_test.go` holds a `deferred` list: every diagnostic lowering is still expected to produce, with the phase that will stop producing it. A corpus case reporting anything else fails the test. Delete entries as phases land; when the list empties, the test becomes a plain assertion that the corpus lowers clean.

## Specification and conformance

`docs/spec.md` is canonical; `docs/grammar.ebnf` holds the formal grammar. Both must be updated alongside any grammar or lexer change.

`testdata/conformance/*/source.tdl` must parse cleanly and lower to the tree in the sibling `ir.golden`; `testdata/invalid/*/source.tdl` must fail with an error containing the text in the sibling `error.golden`. Both corpora are plain text, deliberately not Go code, so a non-Go implementation can run the same checks. `parser/conformance_test.go` walks them automatically, so adding a directory is enough to add a case.

A case directory holding a `pending` file describes a construct the parser cannot read yet and is skipped, with the file's text as the skip reason. The phase that implements the construct deletes the marker. The corpus is the written-down target, not a record of what already works.

`tdl fmt` must be idempotent: formatting canonical output is a no-op.

Every `.tdl` file in `testdata/conformance/` and `prelude/` is stored in canonical form: `tdl fmt <file>` must print it back byte for byte. `examples/*.tdl` deliberately are not, because they carry `//` comments.

The protos are Protobuf Editions 2024. Editions default every field to explicit presence and the generated Go to the opaque API, so each file sets `features.field_presence = IMPLICIT` (what proto3 meant) and `features.(pb.go).api_level = API_OPEN` (the API these types were published with). A field wanting presence says so itself, as `Range.low` does. `go_package` lives in `buf.gen.yaml` under managed mode, not in the files.

After changing `proto/`, run `make generate` and commit `ir/ir.pb.go` with it. Field numbers are a compatibility guarantee to plugins: add fields, never renumber or reuse them. CI enforces that with `buf breaking` against the pull request's base, alongside `buf lint` and `buf format`; `make fmt` formats the protos and `make lint` checks them.

A pull request that has to break the schema carries the `buf skip breaking` label, which is what `bufbuild/buf-action` reads. The workflow only re-runs on push, so label first and then push, or the run will still be working from a payload without it.

## Conventions

Whitespace is insignificant and there are no separator rules: an item ends where the next begins. Commas are required inside `<...>`, conformance lists, and list literals, and are not permitted inside `{ }` blocks.

`where` introduces a constraint block; `requires` introduces class constraints on parameters. Both readings of `{` would otherwise collide.

Declaration keywords are reserved. Modifiers and constraint names (`key`, `owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`) are contextual and remain usable as field names.

A reserved word followed by `:` is a field name: `value: T` is a field, and `include Foo` is still an include while `include: Foo` is a field. A contextual modifier followed by `:` is likewise a name, not a modifier.

A class may not declare key fields, so `key` inside a class body is always the requirement. A class says an implementor must have identity, never which field carries it.

A `<...>` argument is a type or a unit. A bare name could be either, so it is recorded as a type reference and the resolver picks by kind; only an operator (`*`, `/`, `^`) or parentheses makes it unambiguously a unit.

Inside a target block a directive name may be a reserved word, since the directive namespace belongs to the backend.

Directive and constraint arguments are parenthesized and comma separated. Both sets are open, so the parser knows no name's arity and an unparenthesized `min 0 max 100` could not be split.

Regex literals are ambiguous with unit division, so the parser calls `lex.RescanRegexAt` when it wants one. Nothing else in the lexer takes context.

`tdl fmt` drops ordinary `//` comments: the lexer skips them and they never reach the AST. Doc comments (`///`) survive. Fixing this needs comment attachment in the parser and has no phase yet. Never run `tdl fmt` over `examples/*.tdl`: it silently deletes their explanatory headers.

## Releases

release-please owns the version. Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, or `CHANGELOG.md`; each release PR rewrites them. Both version strings carry an `x-release-please-version` annotation, which is what makes them update rather than drift.

`specVersion` is not release-please's and must not be given the annotation. It tracks `docs/spec.md`, which moves on its own schedule, and a release that changed no spec text should not claim to have changed the spec.

`CHANGELOG.md` is excluded from treefmt and from markdownlint, because a generated file that a formatter rewrites is a file the next release will fight over.

The manifest starts at `0.0.34`, the last release the legacy implementation made, and the first release is `0.1.0`. That is computed rather than forced: `feat!: rewrite the lexer and parser` is a breaking change, and `bump-minor-pre-major` turns a breaking change on a `0.x` version into a minor bump. Nothing carries a `release-as` override, so nothing has to be removed afterwards.

`release-please` warns that `version.txt` does not exist on every run. The `simple` release type looks for one by default; this repository states its version in the files that read it instead, and the warning is harmless.
