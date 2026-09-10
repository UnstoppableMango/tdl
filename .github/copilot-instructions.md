# Copilot instructions

@../AGENTS.md

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections.
No expressions, no control flow, no runtime.
This repository owns both the specification and the reference implementation, in Go.

`AGENTS.md` in the repository root is the full guide.
Copilot code review reads it directly, and Copilot CLI expands the reference above, so this file does not restate it.

For reviewing a pull request, the `code-review` agent skill in `.github/skills/code-review/` is the thing to read.
It says what CI already enforces and should draw no comment, which files are generated and from what, and the invariants a diff can break without looking wrong.

Two conventions are worth knowing before any of that, because both look like defects.

Markdown prose is one sentence per line, so a diff stays readable.
The renderer wraps it; do not suggest hard-wrapping.

Commit subjects and pull request titles use Conventional Commits prefixes: `feat:`, `fix:`, `chore:`, `deps:`, `docs:`, `ci:`.
