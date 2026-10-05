# Go

Comment on behavior; formatting and lint are covered.

## Diagnostics accumulate

The parser and `internal/sema` report every problem in one pass, each with a position.
A new error path that stops at the first problem, or reports without a position, is a regression.

## Commands return errors

A command in `internal/cli` returns its error rather than printing it; `cmd/tdl` prints it.

## Boundaries

`ir` and `plugin` are public API that third-party backends compile against.
A change to an exported name there is a compatibility question.
`internal/` is free to change.

`internal/sema` touches no filesystem.
Sources arrive through a `Loader`: `FSLoader` for real files, `MapLoader` in tests.
A direct `os.Open` there is a bug.

## Interning

`internal/sema` interns types so that comparing IDs compares types.
Building a type without going through `intern` breaks that everywhere.

The interning key and the display name differ.
`[T]` and `List<T>` are one type written two ways and stay two entries.

## Completeness

A change to lowering or `ir.Dump` needs `go test ./internal/sema -update`.
A change to `internal/gen` needs `go test ./internal/gen -record`.

## Comments

Comments describe the current state.
Flag temporal language ("now", "previously", "recently added") and comments that explain the diff rather than the code.
