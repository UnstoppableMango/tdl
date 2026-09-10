# proto and ir

`proto/` and `ir/` are the public compatibility surface.
Third-party backends compile against them, in-process and over the plugin protocol.

## Field numbers are a promise

A field may be added.
A field number may never be renumbered or reused, and a field may not change type.

CI runs `buf breaking` against the pull request's base.
A change that has to break carries the `buf skip breaking` label, which is what `bufbuild/buf-action` reads.
The workflow only re-runs on push, so the label goes on before the push.

## Editions 2024

Editions default every field to explicit presence and the generated Go to the opaque API.
Each file sets `features.field_presence = IMPLICIT`, which is what proto3 meant, and `features.(pb.go).api_level = API_OPEN`, the API these types were published with.
A field that wants presence says so itself, as `Range.low` does.
A new file missing either option is a bug.

`go_package` lives in `buf.gen.yaml` under managed mode, never in the proto files.

## Completeness

A change to `proto/` is incomplete without the regenerated `.pb.go` committed alongside it.
`make generate` produces both.

## Conventions in the schema

An `ID` is an index paired with a fully qualified name, and which table it indexes is fixed by the field holding it rather than by the ID.
Every `ID` field says which table in a trailing comment.
A new one without that comment is a gap, because nothing else in the schema records the answer.

An index of `-1` means the name did not resolve.
Code reading an `ID` checks `Resolved()` rather than assuming.
