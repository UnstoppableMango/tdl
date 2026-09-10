# Recorded protocol exchanges

Request and response pairs from `docs/design/plugins.md`, kept as protobuf text.

Each `*.tdl` is lowered, handed to the `debug` backend, and the request it produced and the response it returned are written beside it.
`go test ./internal/gen -run TestRecordedExchanges` checks them; `-record` rewrites them.

They exist so an implementation of the protocol in another language has something to replay and compare against.
A Go test asserting a Go backend against itself proves less than a file another implementation can read.

The text form is compared by parsing it rather than byte for byte, because `prototext` output is deliberately unstable across builds.
