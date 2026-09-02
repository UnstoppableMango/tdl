# Generated files

Review the generator and its input, never the output.
A diff in one of these is a consequence, and the question is whether the change that produced it is right.

| Generated | Produced by | From |
| --- | --- | --- |
| `ir/ir.pb.go` | `make generate` | `proto/tdl/ir/v1/ir.proto` |
| `plugin/plugin.pb.go` | `make generate` | `proto/tdl/plugin/v1/plugin.proto` |
| `tree-sitter/grammar.js` | `go test ./internal/treesitter -update` | `docs/grammar.ebnf` |
| `tree-sitter/src/` | `tree-sitter generate` | `tree-sitter/grammar.js` |
| `testdata/conformance/*/ir.golden` | `go test ./internal/sema -update` | `internal/sema` |
| `testdata/plugin/*.txtpb` | `go test ./internal/gen -record` | `internal/gen` |
| `nix/gomod2nix.toml` | `make tidy` | `go.mod` |
| `CHANGELOG.md` | release-please | commit subjects |

`tree-sitter/src/scanner.c` is the exception.
It lives beside generated files and is hand-written, so it is reviewed like any other source.

The interesting failure is the opposite direction: an input changed and its output did not.
That is a real finding, because CI catches it for `tree-sitter/` and the goldens but a stale `.pb.go` is easy to miss in a diff.

`prototext` output is deliberately unstable across builds, so a whitespace-only diff in a `testdata/plugin/*.txtpb` is noise rather than a change.
The test parses the recorded text rather than comparing bytes, which is why.

## release-please owns versions

Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, or `CHANGELOG.md`.
Each release pull request rewrites them, and both version strings carry an `x-release-please-version` annotation, which is what makes them update rather than drift.

`specVersion` is not release-please's and must not gain that annotation.
It tracks `docs/spec.md`, which moves on its own schedule, and a release that changed no spec text should not claim to have changed the spec.
