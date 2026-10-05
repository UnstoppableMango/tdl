# Generated files

Review the generator and its input, never the output.

| Generated | Produced by | From |
| --- | --- | --- |
| `ir/ir.pb.go` | `make generate` | `proto/tdl/ir/v1/ir.proto` |
| `plugin/plugin.pb.go` | `make generate` | `proto/tdl/plugin/v1/plugin.proto` |
| `tree-sitter/grammar.js` | `make treesitter` | `docs/grammar.ebnf` |
| `tree-sitter/src/` | `tree-sitter generate` | `tree-sitter/grammar.js` |
| `editors/vscode/syntaxes/tdl.tmLanguage.json` | `make textmate` | `docs/grammar.ebnf` |
| `testdata/conformance/*/ir.golden` | `go test ./internal/sema -update` | `internal/sema` |
| `testdata/plugin/*.txtpb` | `go test ./internal/gen -record` | `internal/gen` |
| `nix/gomod2nix.toml` | `make tidy` | `go.mod` |
| `CHANGELOG.md` | release-please | commit subjects |

`tree-sitter/src/scanner.c` is hand-written; review it like any other source.

The finding worth making is an input that changed without its output.
CI catches this for `tree-sitter/` and the goldens, but a stale `.pb.go` is easy to miss.

`prototext` output is unstable across builds, and the test parses the text rather than comparing bytes, so a whitespace-only diff in `testdata/plugin/*.txtpb` is noise.

## release-please owns versions

Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, `version` in `editors/vscode/package.json` and `package-lock.json`, or `CHANGELOG.md`.

`specVersion` tracks `docs/spec.md` and must not gain an `x-release-please-version` annotation.
`metadata.version` in `tree-sitter/tree-sitter.json` is bumped by hand with a regeneration, because `tree-sitter generate` copies it into `parser.c`.
