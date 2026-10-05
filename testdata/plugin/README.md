# Recorded protocol exchanges

Request and response pairs for the [plugin protocol](../../docs/design/plugins.md), in protobuf text format, for implementations in other languages to replay.

Each `*.tdl` is lowered and sent to the `debug` backend; the request and response are written beside it.
`go test ./internal/gen -run TestRecordedExchanges` checks them, and `-record` rewrites them.
The test parses the text rather than comparing bytes, because `prototext` output is unstable across builds.
