---
name: code-review
description: Review a pull request against this repository's invariants. Use when reviewing changes to TDL, a language for describing domain models that owns both its specification and its reference implementation in Go. Covers what CI already enforces and should draw no comment, which files are generated and from what, and the rules that are not visible in a diff.
---

# Reviewing TDL

TDL describes domain models: entities, values, enums, newtypes, classes, and collections.
No expressions, no control flow, no runtime.
This repository owns the specification and the reference implementation, and the pipeline is one package per stage: `lex`, `parser`, `ast`, `internal/sema`, `ir`, `plugin`.

`AGENTS.md` in the repository root describes the architecture.
This skill is about what to check, which is a different question.

## Order of work

1. Read [do-not-review.md](do-not-review.md) first and drop anything on it. Most of the value of this skill is in the comments it stops.
2. Check whether the diff touches a generated file. [generated.md](generated.md) says which files those are and what produces each. Review the input and the generator, never the output.
3. Check the invariants for the areas the diff touches:
   - `proto/` and `ir/` — [proto.md](proto.md)
   - `docs/spec.md`, `docs/grammar.ebnf`, `docs/notation.ebnf`, `tree-sitter/` — [grammar.md](grammar.md)
   - `testdata/`, `prelude/`, `examples/` — [corpora.md](corpora.md)
   - Go under `lex/`, `parser/`, `ast/`, `internal/`, `ir/`, `plugin/` — [go.md](go.md)
4. Check for incompleteness. A change is often correct in what it touches and wrong in what it leaves behind: a proto edit without the regenerated `.pb.go`, a lowering change without regenerated goldens, a grammar change without the spec.

## What a good comment looks like

Name the invariant, not the preference.
Each rule in the reference files is something this repository decided and wrote down, so a comment can say which decision the change breaks and where it is recorded.

Prefer one comment on the thing that will break to five on things that will not.
A reviewer who reports nothing on a change that breaks nothing has done the job.
