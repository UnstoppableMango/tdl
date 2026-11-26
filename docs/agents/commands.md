# Commands

Prefix `make` with `command`, as [AGENTS.md](../../AGENTS.md) explains.

```shell
go test ./...                      # all tests
go test -race ./...                # what CI runs; the plugin subprocess needs it
go test -short ./...               # skips running external tools on generated code
go test ./parser -run TestConformanceCorpusParses   # a single test
go build ./...

command make build            # nix build .#
command make test             # go test ./...
command make cover            # go test -coverpkg=./... -coverprofile=cover.profile ./... + go tool cover -func
command make play             # watch examples/nested.tdl; FILE=scratch.tdl VIEWS=all to override
command make demo             # render docs/demo/demo.tape with VHS into docs/demo/demo.gif
command make lint             # nix flake check + golangci-lint + buf + markdownlint
command make check            # nix flake check alone, a subset of lint
command make fmt              # nix fmt (treefmt) + buf format
command make update           # nix flake update
command make tidy             # go mod tidy + regenerate nix/gomod2nix.toml
command make generate         # buf generate: proto/ -> ir/ir.pb.go
command make treesitter       # docs/grammar.ebnf -> tree-sitter/grammar.js -> tree-sitter/src/
command make textmate         # docs/grammar.ebnf -> the TextMate grammar and editors/bat/tdl.sublime-syntax
command make features         # testdata/features -> each backend's feature table in docs/
command make vscode-install   # package editors/vscode and install it into a running VS Code
command make vscode-check     # npm ci, then typecheck and biome check editors/vscode
command make vscode-test      # in `nix develop .#vscode`: the extension in a headless VSCodium
command make dotnet-test      # dotnet test dotnet/Tdl.slnx: the .NET plugin SDK and the csharp backend
command make dotnet-deps      # restore dotnet/ and rewrite dotnet/deps.json, the NuGet lock Nix builds from
command make rust-test        # cargo test over rust/: the Rust plugin SDK and its debug port
command make js-test          # npm ci, then js/'s tests under node --test and bun test
command make js-check         # npm ci, then build js/ (typecheck) and biome check it
command make mutate           # in the devShell: gremlins over the core packages; PKG=./lex for one
command make test-treesitter  # the conformance corpus, run by tree-sitter
command make check-treesitter # what CI runs: treesitter + a diff + test-treesitter
```

`generate`, `treesitter`, and `textmate` are file targets, so each reruns only when its inputs are newer.
`tree-sitter/Makefile`, `editors/vscode/Makefile`, and `js/Makefile` hold the targets for their directories; the root delegates to them.
`check-treesitter` passes `-B`, because a fresh checkout gives every file the same mtime and a file target would skip the regeneration the diff is meant to test.
`mutate` runs `tools/mutate`, which runs gremlins once per package with `GOFLAGS=-short`, merges the reports into `mutate.json`, and prints the survivors `mutation/equivalent.txt` does not list; `internal/mutation` reads both.
It mutates `lex`, `parser`, `ast`, `internal/sema`, and `internal/analysis`, and nothing runs it in CI (see `docs/architecture/mutation-testing.md`).
`go run ./tools/mutate -report mutate.json` reprints a report without running gremlins.
Never combine `-race` with `-coverpkg`: every covered statement becomes an atomic the race detector tracks, and the protocompile tests time out.
CI runs them as two steps.
`vscode-install` regenerates the TextMate grammar first, and `install.sh` refuses to run without a bundle built by `make`.
