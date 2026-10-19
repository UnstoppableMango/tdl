# Implementing mutation testing

An implementation plan for [mutation-testing.md](mutation-testing.md).
Phases are ordered by dependency, and each states what makes it done.

Phase 1 is done: the devShell carries gremlins 0.6.0, and its run over `lex` under Go 1.27.2 killed 51 mutants, timed out 16, left 13 alive, and reached 7 no test covers, in two and a half minutes.
Phase 2 is done: a full `make mutate` with four CPUs took eight minutes and left 170 survivors and 226 mutants no test covers, `internal/sema` holding 99 of the survivors; the pull request lists the counts per package.

## Scope

This plan builds `make mutate`, the gremlins package, the exclusion list, and the weekly routine.
It changes no production code and adds nothing to CI.
Killing the first survivors is the routine's work, not a phase's.

## Phase 1: the tool

- unmango/pkgs packages gremlins, in its own pull request there; the devShell adds `inputs'.mangopkgs.packages.gremlins` after `make update` picks it up, and `nix flake check` still passes.
- Confirm gremlins builds and runs under the devShell's Go 1.27, and that it can pass `-short` to `go test` (a flag, or `GOFLAGS`); if neither works, the design's test command is revisited before phase 2.
- Run it once by hand over `lex`, the smallest package in scope, with every mutator enabled.

Done when the devShell runs `gremlins unleash ./lex` to completion, and the pull request reports the wall-clock time and the killed, survived, not-covered, and timed-out counts.

## Phase 2: `make mutate`

- `.gremlins.yaml` enables every mutator and keeps the default timeout coefficient.
- `make mutate` runs gremlins over the five packages in scope, or over `PKG`, writing `mutate.json`; `mutate.json` joins `.gitignore`.
- The target prints the survivors one per line as `file:line:column mutator`, then the not-covered mutants under their own heading, then the counts per package.
- `mutation/equivalent.txt` exists, empty but for a comment stating its format; the target drops listed mutants from the survivors and prints a listed position that names no mutant as stale.
- `AGENTS.md` gains the target in its command list and a line on what it mutates.

Done when `command make mutate` completes a full run in the devShell, the pull request reports its wall-clock time and counts per package, and adding a survivor's line to `mutation/equivalent.txt` removes it from the output.

## Phase 3: the routine

- A Claude routine runs on Thursdays in its own thread: a full `make mutate`, falling back to diff mode against the previous run's commit if the full run measured in phase 2 does not fit a session.
- Per package with survivors, at most one pull request adding the tests that kill up to ten, with the survivor counts before and after in its description.
- An issue for each survivor that is a real bug, dead code, or equivalent, the last proposing its line for `mutation/equivalent.txt`.
- The counts per package are recorded in project memory after each run.

Done when the routine's first run has fired, opened its pull requests and issues, and recorded its counts.
