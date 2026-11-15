# AGENTS.md

Guidance for coding agents working in this repository.

TDL is a language for describing domain models; this repository holds its specification (`docs/spec.md`), its reference implementation in Go, and plugin SDKs in .NET, Rust, and TypeScript.

## Build and test

```shell
go build ./...
go test -race ./...   # what CI runs; the plugin subprocess needs it
go test -short ./...  # skips running external tools on generated code
command make lint     # nix flake check + golangci-lint + buf + markdownlint
command make fmt      # nix fmt (treefmt) + buf format
```

Prefix `make` with `command`: zsh with Prezto shadows it with an autoload stub that fails in an agent's subprocess.
Nix builds the project, and `nix flake check` fails on anything unformatted.

After changing `go.mod`, run `make tidy` so `nix/gomod2nix.toml` stays in sync; otherwise `nix build` fails.
Renovate does the same for its own updates through the `gomod2nix` preset in `.github/renovate.json`.
Neither `gomod2nix` nor `protoc-gen-go` is on `PATH`.
Run generators through the devShell: `nix develop --command make tidy`, and `buf generate` inside `nix develop` or with `PATH="$(go env GOPATH)/bin:$PATH"`.

## Every change

- A generated file changes only through its input and generator; `.github/skills/code-review/generated.md` lists them.
- When code a `docs/architecture/` page describes changes, edit the page in the same pull request.
- release-please owns every version number; [releases.md](docs/agents/releases.md) lists the two exceptions.

## Read before working on

| Area | File |
| --- | --- |
| The full command list, `make` targets, mutation testing | [commands.md](docs/agents/commands.md) |
| Formatters, linters, and what each covers | [formatting.md](docs/agents/formatting.md) |
| `flake.nix`, `nix/`, packages, checks, Hercules CI | [nix.md](docs/agents/nix.md) |
| The front end and middle: `lex`, `parser`, `ast`, `internal/sema`, `ir`, `internal/analysis`, `internal/gen` | [architecture.md](docs/agents/architecture.md) |
| A backend, `backend/internal/emit`, round trips, the feature tables | [backends.md](docs/agents/backends.md) |
| The .NET, Rust, or JS plugin SDK | [sdks.md](docs/agents/sdks.md) |
| `internal/cli` or `internal/lsp` | [cli.md](docs/agents/cli.md) |
| `docs/spec.md`, the grammars, tree-sitter, TextMate, VS Code | [grammar.md](docs/agents/grammar.md) |
| `proto/` and generated protobuf code | [protobuf.md](docs/agents/protobuf.md) |
| Goldens and the conformance and invalid corpora | [testing.md](docs/agents/testing.md) |
| TDL syntax rules: separators, reserved words, comments | [language.md](docs/agents/language.md) |
| Specs, tickets, ADRs, review, CI, stacked pull requests | [workflow.md](docs/agents/workflow.md) |
| Releases and published packages | [releases.md](docs/agents/releases.md) |
| graphify setup and the code graph | [knowledge-graph.md](docs/agents/knowledge-graph.md) |

## Agent skills

### Issue tracker

Issues are tracked in this repository's GitHub Issues. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default triage labels, each named for its role. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one root `CONTEXT.md` and `docs/adr/`, whose records start at ADR-0001 with no backfill. See `docs/agents/domain.md`.

### Weekly routines

Code reduction (Mondays) and mutation testing (Thursdays) file `needs-triage` issues with a `Kind:` line and build the `ready-for-agent` ones of their kinds.
Code reduction opens a pull request only when it shrinks the code not marked `linguist-generated` in `.gitattributes`, and keeps its ranked targets in the pinned `routine:targets` issue. See `docs/agents/routines.md`.
