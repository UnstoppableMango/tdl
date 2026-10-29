# Mutation testing

Design document.
`make mutate` is built and the weekly routine runs it every Thursday.

Coverage says a line ran; it does not say a test would notice the line being wrong.
Mutation testing changes the code one small edit at a time, a `<` to a `<=`, an `&&` to an `||`, a dropped `continue`, and runs the tests against each mutant.
A mutant the tests fail on is killed.
One they pass on survives, and a survivor names a behavior no test pins down.

## Goal

Find the assertions the core is missing, and add them as tests.
The measure is the number of survivors, which should fall; nothing gates on it.
A score threshold is deliberately absent, because it rewards tests written to move a number rather than to kill a specific mutant.

## Scope

Mutants are made only in the packages where a wrong answer silently changes every target's output:

- `lex`
- `parser`
- `ast`
- `internal/sema`
- `internal/analysis`

No backend is mutated, `backend/internal/emit` included.
A backend's output is checked by goldens and by its language's own compiler, which a mutant test cannot improve on, and most of what `emit` computes it reads from `internal/analysis`, which is in scope.
Generated code (`*.pb.go`, `tree-sitter/src/`), `cmd/`, `internal/cli`, and `internal/lsp` are also out.

A mutant runs only the tests of the package it is in, gremlins' default, and never a backend's.
A mutant in `internal/sema` that only a backend test would notice survives, which is the point: lowering's own tests should pin what lowering computes.

## Tool

[gremlins](https://github.com/go-gremlins/gremlins) makes and runs the mutants.
It is coverage guided: it first runs the tests with coverage and skips a mutant no test reaches, reporting it as not covered rather than spending a test run on it.
A not-covered mutant is a coverage gap, not a survivor, and is reported separately.

Every mutator is enabled, not only gremlins' defaults:

| Mutator | Edit |
| --- | --- |
| `arithmetic-base` | `+` to `-`, `*` to `/`, ... |
| `conditionals-boundary` | `<` to `<=`, ... |
| `conditionals-negation` | `==` to `!=`, `<` to `>=` |
| `increment-decrement` | `++` to `--` |
| `invert-assignments` | `+=` to `-=`, ... |
| `invert-bitwise` | `&` to `\|`, ... |
| `invert-bwassign` | `&=` to `\|=`, ... |
| `invert-logical` | `&&` to `\|\|` |
| `invert-loopctrl` | `break` to `continue` |
| `invert-negatives` | `-x` to `+x` |
| `remove-self-assignments` | `x += y` to `x = y` |

The lexer, the parser, and lowering are full of short-circuit guards and early `continue`s, which are what the extra mutators reach.

gremlins is not in nixpkgs, so [unmango/pkgs](https://github.com/unmango/pkgs) packages it, and the devShell takes it from the `mangopkgs` input as it takes graphify.

## Running

`make mutate` is the whole of the repository's machinery:

```shell
command make mutate              # every package in scope
command make mutate PKG=./lex    # one
```

Each mutant runs `go test -short` without `-race`.
gremlins has no flag for `-short`, so the target sets `GOFLAGS=-short`, which `go test` reads, for the coverage run and every mutant alike.
`-short` skips running go, smithy, tsc, ghc, dotnet, and likec4 on generated code, which would make each mutant take minutes, and `-race` multiplies the run time while catching nothing a mutant introduces.
A mutant is timed out, and counted as killed, at a multiple of the package's unmutated test time; gremlins' default coefficient stands unless the first full run shows it misjudging.

A mutant is killed when any test fails, a golden comparison included.
The goldens (`ir.golden`, `error.golden`, the recorded plugin exchanges) are the conformance contract, so a mutant that moves one is a behavior change the tests caught.

gremlins names a report's files relative to the package it ran in, so the target runs it once per package and writes the reports merged, with file names relative to the module root, to `mutate.json`, which is gitignored, and prints the survivors after the exclusion list removes the known equivalents.
Nothing runs it in CI.

## Equivalent mutants

Some mutants cannot change behavior: `i < n` to `i != n` in a loop whose index never passes `n`, or a boundary flipped on a comparison only ever made between unequal values.
No test can kill one.

`mutation/equivalent.txt`, checked in, lists them one per line as the report identifies them, a file, line, column, and mutator, with a comment saying why the mutant is equivalent:

```text
# The loop index starts at 0 and steps by 1, so it reaches n exactly.
lex/lexer.go:212:14 conditionals-boundary
```

`make mutate` drops a listed mutant from the survivors it prints.
The source carries no annotations for the tool.
A listed position that no longer names a mutant is printed as stale, so an edit that moves or removes the code shows up, and the weekly code-reduction routine may prune the list as it prunes code.

## The weekly routine

`docs/agents/routines.md` holds the routine's steps and how its issues move through triage; this section is why it works that way.

A Claude routine runs `make mutate` every Thursday, a full run over every package in scope.
Thursday keeps it off Monday, when the code-reduction routine changes code it would otherwise race.
A full run revisits a survivor whose code did not change, which a run over only the changed code never does; if a full run proves too slow for a weekly session, the routine falls back to gremlins' diff mode against the previous run's commit.

For each package with survivors it opens at most one pull request, holding the tests that kill up to ten of them; the rest wait for the next week.
One pull request per mutant would flood the stack, and one for everything would be unreviewable.

The routine's pull requests add or extend tests and change no production code.
What it finds that a test cannot fix becomes an issue:

- **A real bug.** The mutant's behavior is the right one, or neither is; the fix goes through the normal path.
- **Dead code.** No input reaches the mutated line's other outcome; the code-reduction routine is the one to remove it.
- **An equivalent mutant.** The issue proposes the line for `mutation/equivalent.txt`; once triaged `ready-for-agent`, the routine's next run adds it, after checking its own report still lists that mutant as lived and the reasoning still holds, since a position names whatever mutant sits there now.

Each pull request's description quotes the survivor count per package, before and after.
The routine also records the counts in project memory, so the trend is visible without a checked-in score file that would churn every week.

## Not covered

- **Backends.** Out of scope until the core's survivors are few; a later design may add one backend at a time.
- **Gating.** No check fails on a mutant, and no pull request is held for one.
- **Fuzzing.** A different question, whether some input breaks the code, rather than whether the tests would notice a change.
