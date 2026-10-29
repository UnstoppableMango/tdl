# Weekly routines

Two Claude routines run against this repository every week, each from its own project thread.
Each routine's prompt is one line naming its section here, so a change to a routine is a pull request against this file.
A routine's prompt can be edited only from the thread it posts into.

| Routine | When | Section |
| --- | --- | --- |
| Weekly code reduction | Mondays, 08:45 UTC | [Code reduction](#code-reduction) |
| Weekly mutation testing | Thursdays, 08:38 UTC | [Mutation testing](#mutation-testing) |

Thursday keeps mutation testing off Monday, when code reduction changes the code it would otherwise race.

## Every run

1. Re-read `AGENTS.md`, `CONTEXT.md`, and `docs/agents/domain.md`, plus the architecture doc the routine's section names.
1. Drive the open pull requests from earlier runs (CI, reviews) before opening new ones; the section says which count against its cap.
1. Work the [queue](#the-queue) before the routine's own sweep.
1. Fetch and branch from the latest default branch for every pull request.
1. Reply in the routine's thread with the links to every pull request and issue the run opened, or one line saying it opened none.

Neither routine invokes a skill by name, since a routine session may not load them; each follows the conventions in `docs/agents/` as written.

### Domain docs

Name every concept the way `CONTEXT.md` does, in issue titles, test names, commit messages, and pull request text, and avoid the words it lists to avoid.
Never edit `CONTEXT.md` or write an ADR: a routine records no term and makes no decision.
A change that would contradict an ADR, or a decision recorded in `docs/spec.md`, the Conventions in `AGENTS.md`, or `docs/architecture/`, is skipped and filed as an issue instead.

### Filing an issue

Search open and closed issues for a duplicate first, and file nothing if one exists.

Each issue has:

- a plain title in the glossary's words;
- a `Kind:` line, the first line of the body, naming one kind from the [table](#kinds);
- what was seen, and where, as `file:line` on the commit read;
- a minimal reproduction, or the mutant as `file:line:column mutator`, when there is one;
- why it might matter;
- a closing line saying which routine filed it and that it is unverified.

Label it `needs-triage`, and also `bug` when its kind is `bug`.
Never apply a `ready-for-*` label, create a label, or judge the issue at length: triage decides.
Assign UnstoppableMango.

### Kinds

The kind decides which routine builds the issue once it is triaged, whichever routine filed it.

| Kind | Filed when | Built by |
| --- | --- | --- |
| `bug` | a suspected bug, a missing check, an input that panics or is wrongly accepted, or a mutant whose behavior is the right one | neither: a thread Erik starts |
| `dead-code` | no input reaches the code, or a mutated line's other outcome | code reduction |
| `duplicate-code` | two copies a reduction would merge disagree | code reduction |
| `equivalent-mutant` | no test can tell the mutant apart; the issue proposes the exact `mutation/equivalent.txt` line and its comment | mutation testing |
| `other` | anything else that looks wrong | neither: a thread Erik starts |

### The queue

A routine's queue is the open issues whose `Kind:` line names a kind it builds:

- **`ready-for-agent`**: build it, in its own pull request whose description says `Closes #N`, before the sweep.
- **`needs-info`**: answer the question in a comment from what the code shows, then swap the label back to `needs-triage`.
- **`needs-triage`**, **`ready-for-human`**, **`wontfix`**: leave it alone.

A queue pull request counts against the routine's cap as the sweep's do, and follows the same rules.
A queue issue whose fix would break those rules, such as a `dead-code` issue whose removal changes output, gets a comment saying why and goes back to `needs-triage`.

## Code reduction

The job makes the codebase smaller and simpler, because less code is easier to maintain.
Behavior stays identical.

Cap: one open pull request, on a branch named `chore/reduce-code-YYYY-MM-DD`.
If an earlier run's is still open, drive it and stop.
Queue issues go first, and the sweep fills the same pull request, which lists `Closes #N` for each.

1. Find code to remove or simplify: dead or unreachable code; unused functions, types, constants, and parameters; duplicated logic that can share one helper; needless indirection, such as single-use wrappers or one-implementation interfaces that buy nothing; verbose code with a shorter equivalent; stale comments; positions in `mutation/equivalent.txt` that `make mutate` prints as stale.
   Use `go vet`, golangci-lint's unused checks, `deadcode` (`golang.org/x/tools/cmd/deadcode`), and `dupl` where available.
1. Keep to these rules:
   - Behavior is identical: no change to CLI output, generated backend output, the IR, the exported API of `ir/` and `plugin/`, or `proto/`.
   - Never remove a test case.
     A test may be simplified (table-driven, shared helpers, less setup) as long as every case still runs and asserts the same thing.
   - Never hand-edit generated files, goldens, release-please versions, or `CHANGELOG.md`, and never remove spec content or conformance cases.
   - Prefer several small, obviously safe reductions to one risky rewrite, and skip anything not clearly behavior-preserving.
   - When code moves into a shared package such as `backend/internal/emit`, add a small table test there, since coverage is measured per package.
1. File an issue for every hole found along the way, and fix none of them in the reduction pull request.
1. Verify with `go build ./...`, `go test -race ./...`, and `golangci-lint run`, plus `nix fmt` and `nix flake check` when nix runs.
   Goldens pass without `-update`.
   A reduction that breaks anything is reverted, never fixed by changing a test.
1. Commit as a conventional commit (`refactor: remove unused helpers`) and open the pull request.
   Its description has "Before:" and "After:" paragraphs, each reduction and why it is safe, the net line change from `git diff --shortstat`, the `Closes #N` lines, and the issues filed this run.
   Assign UnstoppableMango, subscribe to its activity, and drive CI to green.

## Mutation testing

The job finds behaviors the core's tests do not pin down and adds the tests that do, as `docs/architecture/mutation-testing.md` describes.
It never changes production code.

Cap: one open pull request per package, on a branch named `test/mutation-<package>-YYYY-MM-DD`, the package path with `/` replaced by `-`.
A package with an open pull request from an earlier run is driven and left out of this run.

1. Run `nix develop --command make mutate` on the latest default branch, timing it.
   If a full run does not fit in the session, fall back to gremlins' diff mode (`--diff`) against the commit the previous run recorded in memory, and say so in the thread.
1. For each package with survivors and no open pull request, add tests that kill up to ten of them.
   Prefer survivors in code a user's model goes through, and extend existing table tests over adding files.
   - Tests only: never edit production code, goldens (unless a test needs a new golden case), or generated files.
     `mutation/equivalent.txt` changes only for a `ready-for-agent` `equivalent-mutant` issue, by the line it proposes, once this run's report still lists that mutant as lived and the issue's reasoning still holds for the code at that position.
   - Confirm the kills with `nix develop --command make mutate PKG=./<package>`; a test that kills nothing is fixed or dropped.
   - Verify with `go test -race ./<package>/...` and `golangci-lint run ./<package>/...`, plus `nix fmt` when it runs.
1. Commit as `test(<package>): kill N mutation survivors` and open the pull request.
   Its description has "Before:" and "After:" paragraphs, the package's survivor count before and after, and the killed mutants as `file:line:column mutator`.
   Assign UnstoppableMango, subscribe to its activity, and drive CI to green.
1. File an issue for each survivor a test cannot kill: `bug`, `dead-code`, or `equivalent-mutant`.
   A not-covered mutant is a coverage gap, not a survivor: it is counted in the run's record and gets neither a test nor an issue.
1. Record the run in the project memory file `mutation-testing.md`: date, commit, wall-clock time, and per package killed, lived, timed out, and not covered, keeping the last eight runs.
1. In the thread reply, compare each package's survivor count with the previous run.
