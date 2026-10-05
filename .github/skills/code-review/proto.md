# proto and ir

`proto/` and `ir/` are the public compatibility surface for third-party backends.

## Field numbers are a promise

Fields may be added.
A field number is never renumbered or reused, and a field never changes type.

`buf breaking` runs in `.github/workflows/buf.yml`.
It does not block a merge, so a failing `buf` check without the `buf skip breaking` label is worth a comment.

## Editions 2024

Each file sets `features.field_presence = IMPLICIT` and `features.(pb.go).api_level = API_OPEN`.
A new file missing either is a bug.
A field that needs presence sets it itself, as `Range.low` does.

`go_package` lives in `buf.gen.yaml`, never in a proto file.

## Completeness

A change to `proto/` ships with the `.pb.go` that `make generate` produces.

## IDs

An `ID` is an index plus a fully qualified name; the field holding it decides which table it indexes.
Every `ID` field names its table in a trailing comment, and a new one without that comment is a gap.
An index of `-1` means unresolved, so code checks `Resolved()`.
