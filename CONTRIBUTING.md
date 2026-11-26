# Contributing

## Issues

File a bug or a request in [GitHub Issues](https://github.com/UnstoppableMango/tdl/issues).
Report a security problem privately instead, as [SECURITY.md](SECURITY.md) says.

A new issue starts as `needs-triage`.
Triage moves it to one of the other four labels:

- `needs-info`: waiting on the reporter for more information.
- `ready-for-agent`: fully specified, so a coding agent can build it.
- `ready-for-human`: fully specified, but a person needs to build it.
- `wontfix`: will not be done.

[docs/agents/triage-labels.md](docs/agents/triage-labels.md) maps these to the roles the agent skills use.

## Pull requests

- A pull request is squash merged, and its title becomes the commit's, so write it as a [conventional commit](https://www.conventionalcommits.org/): `feat(lex): ...`, `fix(gen): ...`, `docs: ...`.
- The `required` check must pass.
- Every review thread must be resolved: reply with what changed, or why nothing did, then resolve it.
- Keep a pull request small; split a large change into a stack of smaller ones.

## Tests

New functionality comes with tests.
A bug fix comes with a test that fails without the fix.

## Commands

```shell
go test ./...         # all tests
command make lint     # nix flake check + golangci-lint + buf + markdownlint
command make fmt      # nix fmt + buf format
```

[AGENTS.md](AGENTS.md) has every other command and the architecture.
