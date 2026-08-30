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

TDL is a type description language: records, enums, primitives, collections. No expressions, no control flow, no runtime. This repo owns both the specification and the reference implementation.

Pipeline, one package per stage:

- `lex` — hand-written lexer. `lex.Kind` covers idents, literals, keywords, and punctuation; `LookupIdent` turns an identifier into a keyword kind. Positions originate here and flow through the AST as `ast.Position` (a type alias).
- `parser` — recursive descent over the token stream, producing `*ast.File`. Errors accumulate in an `ErrorList` rather than aborting: `syncTop` and `syncField` resynchronize at declaration and field boundaries so one bad line does not swallow the rest of the file.
- `ast` — parse tree mirroring source 1:1, names left unresolved. `ast.Fprint` produces the canonical formatting used by `tdl fmt`.
- `internal/cli` — cobra commands (`ast`, `check`, `fmt`, `play`, `tokens`, `version`) wired in `root.go`. `play` is a watch-mode playground that re-renders a file on save; `examples/` holds files to experiment with and is outside the conformance corpus.
- `cmd/tdl` — main.

An `ir` package (resolved semantic model consumed by backends) is referenced in doc comments but does not exist yet. There are no code-generation backends.

## Specification and conformance

`docs/spec.md` is canonical; `docs/grammar.ebnf` holds the formal grammar. Both must be updated alongside any grammar or lexer change.

`testdata/conformance/*/source.tdl` must parse cleanly; `testdata/invalid/*/source.tdl` must fail with an error containing the text in the sibling `error.golden`. Both corpora are plain text, deliberately not Go code, so a non-Go implementation can run the same checks. `parser/conformance_test.go` walks them automatically, so adding a directory is enough to add a case.

`tdl fmt` must be idempotent: formatting canonical output is a no-op.

## Conventions

The spec documents deliberate M1 simplifications, not bugs. Notably: keywords are rejected as annotation argument names (`@go(package: "x")` is a syntax error), and `union` plus generic type parameters are reserved in the grammar but unimplemented. Reserving them keeps their later addition additive.

`toolVersion` and `specVersion` are hardcoded constants in `internal/cli/version.go`.
