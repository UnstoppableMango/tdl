# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## Commands

```shell
go test ./...                      # all tests
go test ./parser -run TestConformanceCorpusParses   # a single test
go build ./...

command make build     # nix build .#
command make test      # go test ./...
command make play      # watch scratch.tdl; FILE=examples/nested.tdl VIEWS=all to override
command make lint      # nix flake check (golangci-lint + treefmt)
command make fmt       # nix fmt (treefmt: gofmt, nixfmt, actionlint)
command make tidy      # go mod tidy + regenerate nix/gomod2nix.toml
```

Prefix `make` with `command` (see the shell autoload note in the global instructions).

After changing `go.mod` or adding dependencies, run `make tidy` so `nix/gomod2nix.toml` stays in sync, otherwise `nix build` fails.

## Architecture

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections. No expressions, no control flow, no runtime. This repo owns both the specification and the reference implementation.

The specification is ahead of the implementation. `docs/spec.md` and the corpus describe the current grammar; the parser still reads the M1 grammar and cannot parse them. `docs/design/` holds the plans that close the gap, and `docs/design/parser-plan.md` is the head of the queue.

Pipeline, one package per stage:

- `lex` — hand-written lexer. `lex.Kind` covers idents, literals, keywords, and punctuation; `LookupIdent` turns an identifier into a keyword kind. Positions originate here and flow through the AST as `ast.Position` (a type alias). Regex literals are scanned only on request via `RescanRegexAt`, because `/` is also division in a unit expression.
- `parser` — recursive descent over the token stream, producing `*ast.File`. Errors accumulate in an `ErrorList` rather than aborting: `syncTop` resynchronizes at the next declaration so one bad line does not swallow the rest of the file.
- `ast` — parse tree mirroring source 1:1, names left unresolved. `ast.Fprint` produces the canonical formatting used by `tdl fmt`.
- `internal/cli` — cobra commands (`ast`, `check`, `fmt`, `play`, `tokens`, `version`) wired in `root.go`. `play` is a watch-mode playground that re-renders a file on save; `examples/` holds files to experiment with and is outside the conformance corpus.
- `internal/sema` — ast to ir. Does not exist yet; see `docs/design/ir-plan.md`.
- `prelude/std.tdl` — the standard prelude, written in TDL. Nothing loads it yet; it is the target the parser is built against.
- `cmd/tdl` — main.

An `ir` package (resolved semantic model consumed by backends) is designed in `docs/design/ir.md` but does not exist yet. There are no code-generation backends.

## Specification and conformance

`docs/spec.md` is canonical; `docs/grammar.ebnf` holds the formal grammar. Both must be updated alongside any grammar or lexer change.

`testdata/conformance/*/source.tdl` must parse cleanly; `testdata/invalid/*/source.tdl` must fail with an error containing the text in the sibling `error.golden`. Both corpora are plain text, deliberately not Go code, so a non-Go implementation can run the same checks. `parser/conformance_test.go` walks them automatically, so adding a directory is enough to add a case.

A case directory holding a `pending` file describes a construct the parser cannot read yet and is skipped, with the file's text as the skip reason. The phase that implements the construct deletes the marker. The corpus is the written-down target, not a record of what already works.

`tdl fmt` must be idempotent: formatting canonical output is a no-op.

## Conventions

Whitespace is insignificant and there are no separator rules: an item ends where the next begins. Commas are required inside `<...>`, conformance lists, and list literals, and are not permitted inside `{ }` blocks.

`where` introduces a constraint block; `requires` introduces class constraints on parameters. Both readings of `{` would otherwise collide.

Declaration keywords are reserved. Modifiers and constraint names (`key`, `owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`) are contextual and remain usable as field names.

A reserved word followed by `:` is a field name: `value: T` is a field, and `include Foo` is still an include while `include: Foo` is a field. A contextual modifier followed by `:` is likewise a name, not a modifier.

Inside a target block a directive name may be a reserved word, since the directive namespace belongs to the backend. Directive arguments are parenthesized and comma separated.

`union` is reserved in the grammar and unimplemented. Reserving it keeps its later addition additive.

`tdl fmt` drops ordinary `//` comments: the lexer skips them and they never reach the AST. Doc comments (`///`) survive. Fixing this needs comment attachment in the parser and has no phase yet.

`toolVersion` and `specVersion` are hardcoded constants in `internal/cli/version.go`.
