# Workflow

## Docs

`docs/backends/` holds a page per backend for model authors (see [Feature tables](backends.md#feature-tables)).
`docs/architecture/` is reference: what a built area is meant to be, and why.
Read the one covering an area before changing it, and edit it in the same pull request when the code it describes changes.
New work goes through the skills and the issue tracker:

- a feature or backend: a spec issue (`/to-spec`), broken into ticket issues with blocked-by links (`/to-tickets`);
- a decision that is hard to reverse: an ADR in `docs/adr/` (`/domain-modeling`);
- a term: `CONTEXT.md`.

A phase's pull request closes its ticket.
A spec issue's sub-issues are its phases, and an open ticket is unbuilt work, so write a plan as tickets, never as a `*-plan.md`.
Wanted work with no spec is a `needs-triage` issue.
`CONTEXT.md` is the glossary: one canonical word per concept, and the words to avoid.

## Review

Copilot code review reads `AGENTS.md`, which links here.
`.github/skills/code-review/` holds the review procedure and per-area invariants; `do-not-review.md` lists what CI enforces.
`.github/copilot-instructions.md` is always loaded and stays short.
Its `@../AGENTS.md` is expanded by Copilot CLI; skill files expand no references and stand alone.

CodeRabbit reviews only the tip of a stack.
A lower pull request with no comments may be unreviewed; check the CodeRabbit check, not the thread count.

DeepSource (`.deepsource.toml`) runs the `go`, `shell`, `secrets`, and `test-coverage` analyzers.
Coverage comes from the `check` job's `cover.profile`, authenticated with OIDC; the CLI comes from the devShell.
Check a Go style finding against `.golangci.yml` before acting on it.

`main` requires every review thread resolved and no approval.

The one required check is the `required` job in `.github/workflows/ci.yml`.
It needs `check`, on `ubuntu-latest`, and `darwin`, on `macos-latest`.
A new CI job goes in its `needs`, not in the ruleset (declared in Pulumi in `UnstoppableMango/vcs`).

Pull requests are stacked, and GitHub rebases the rest of the stack on merge; reset local branches from the remote afterwards.
`gh pr merge` and `PUT /pulls/{n}/merge` refuse a stacked pull request.
Use `PUT /pulls/{n}/merge-async`, which returns `{"status":"pending"}` and merges a few seconds later.
