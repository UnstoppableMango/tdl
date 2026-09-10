# Do not review

CI enforces all of this, and a comment about any of it is noise.

- **Formatting.** `nix fmt` runs treefmt over Go, Nix, YAML, JSON, TOML, Markdown, and protobuf. `nix flake check` fails when anything is unformatted, so an unformatted file cannot reach a review.
- **Go lint.** `.golangci.yml` enables `misspell`, `unconvert`, `unparam`, `gofmt`, and `goimports`.
- **Markdown style.** `markdownlint-cli2` runs over what `.markdownlint-cli2.yaml` lists.
- **Protobuf style.** `buf lint` and `buf format` run in CI, and `buf breaking` runs against the pull request's base.
- **Line length, import order, and naming conventions.** A linter owns these or the repository has decided not to care.

Two conventions look like defects and are not.

Markdown prose is one sentence per line, so a diff stays readable.
The renderer wraps it. Never suggest hard-wrapping or joining lines.

`docs/grammar.ebnf` and `docs/notation.ebnf` have no formatter, and their column alignment is chosen per section for reading.
Alignment there is deliberate.
