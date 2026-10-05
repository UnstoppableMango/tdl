# Do not review

CI enforces these, so a comment on any of them is noise:

- **Formatting.** `nix fmt` covers Go, Nix, YAML, JSON, TOML, Markdown, protobuf, and TypeScript, and `nix flake check` fails on unformatted files.
- **Go lint.** `golangci-lint` with the linters in `.golangci.yml`.
- **Markdown style.** `markdownlint-cli2` over the files `.markdownlint-cli2.yaml` lists.
- **Protobuf style.** `buf lint`, `buf format`, and `buf breaking`.
- **Line length, import order, naming.** A linter owns these or the repository does not care.

Two conventions look like defects and are not:

- Markdown prose is one sentence per line. Never suggest hard-wrapping or joining lines.
- `docs/grammar.ebnf` and `docs/notation.ebnf` have no formatter, and their column alignment is deliberate.

Two kinds of finding need evidence:

- **Pre-existing behavior.** Check the base branch first. A bug the base already has belongs in an issue, not in this pull request; say which it is.
- **Performance and allocation.** Back the claim with `go test -bench` and `-benchmem`. Reading the source is a guess, and a wrong guess sends the fix to the wrong line.
