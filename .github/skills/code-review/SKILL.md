---
name: code-review
description: Review a pull request against this repository's invariants. Use when reviewing changes to TDL, a language for describing domain models that owns both its specification and its reference implementation in Go. Covers what CI already enforces and should draw no comment, which files are generated and from what, and the rules that are not visible in a diff.
---

# Reviewing TDL

`AGENTS.md` describes the architecture.
This skill says what to check.

## Order of work

1. Read [do-not-review.md](do-not-review.md) and drop anything it lists.
2. If the diff touches a generated file, review its input and generator instead. [generated.md](generated.md) lists them.
3. Check the invariants for the areas the diff touches:
   - `proto/` and `ir/`: [proto.md](proto.md)
   - `docs/spec.md`, `docs/grammar.ebnf`, `docs/notation.ebnf`, `tree-sitter/`: [grammar.md](grammar.md)
   - `testdata/`, `prelude/`, `examples/`: [corpora.md](corpora.md)
   - Go code: [go.md](go.md)
4. Check what the change left behind: a proto edit without the regenerated `.pb.go`, a lowering change without regenerated goldens, a grammar change without the spec, a deleted file with links still pointing at it.
   Nothing checks that a link resolves, so grep for references to a deleted or renamed file.

## Writing comments

Name the invariant the change breaks and where it is written down.
One comment on something that will break beats five on things that will not.
Reporting nothing on a change that breaks nothing is a correct review.
