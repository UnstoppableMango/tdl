# Protobuf

The protos are Editions 2024.
Each file sets `features.field_presence = IMPLICIT` and `features.(pb.go).api_level = API_OPEN`; a field needing presence sets it itself, as `Range.low` does.
`go_package` lives in `buf.gen.yaml` under managed mode.

After changing `proto/`, run `command make generate` and commit the `.pb.go` files, `dotnet/Tdl.Plugin/Generated/`, `rust/tdl-plugin/src/generated/`, and `js/tdl-plugin/src/generated/`.
`checks.generate` runs `buf generate` in the sandbox with the devShell's plugins and fails on any difference from what is committed.
Field numbers are a compatibility promise to plugins: add fields, never renumber or reuse.
`.github/workflows/buf.yml` runs `buf lint`, `buf format`, and `buf breaking` against the base.
It is advisory: it is outside the `required` job, so a reviewer must read it.
A pull request that must break the schema carries the `buf skip breaking` label; toggling it reruns the check.
