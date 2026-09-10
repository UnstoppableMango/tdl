# Go

Formatting and lint are covered elsewhere. Comment on behavior.

## Diagnostics accumulate

The parser and `internal/sema` report every problem in one pass, each with a position.
`parser` collects into an `ErrorList` and `syncTop` resynchronizes at the next declaration, so one bad line does not swallow the rest of the file.

A new error path that returns on the first problem, or that reports without a position, is a regression.

A form the compiler does not handle yet reports a diagnostic naming the deferral rather than dropping the input silently.

## Commands return errors

A command under `internal/cli` returns an error rather than printing it.
The root silences cobra's error printing so a diagnostic list renders as itself, and `cmd/tdl` prints whatever a command returns.

## Boundaries

`ir` and `plugin` are public API that third-party backends compile against, so a change to an exported name there is a compatibility question.
`internal/` is private and free to change.

`internal/sema` touches no filesystem.
Imported sources arrive through a `Loader`, with `FSLoader` for real files and `MapLoader` for tests.
A direct `os.Open` in that package breaks the tests' ability to supply sources.

## Interning

`internal/sema` interns the type table, which is what makes an ID comparison a type comparison.
A change that adds a way to build a type without going through `intern` breaks that property everywhere.

The interning key and the display name are different things.
The key separates `[T]` from `List<T>`, which are the same type written two ways and must stay two entries; the name is what a person reads.

## Completeness

Changing lowering or `ir.Dump` means regenerating goldens with `go test ./internal/sema -update`.
Changing `internal/gen` means `go test ./internal/gen -record`.
A behavior change arriving without them is incomplete, and the reviewer sees it as a test that did not run.

## Comments

Doc comments describe the current state, as if it had always been that way.
Flag temporal or narrative language: "now", "previously", "this was changed to", "recently added", or a comment that explains the diff rather than the code.
It rots, and the next change has to clean it up.
