# Recorded protocol exchanges

Request and response pairs for the [plugin protocol](../../docs/architecture/plugins.md), for implementations in other languages to replay.

Each `*.tdl` is lowered and sent to the `debug` backend; the request and response are written beside it, as protobuf text (`*.txtpb`) and in the wire encoding (`*.binpb`).
`go test ./internal/gen -run TestRecordedExchanges` checks them, and `-record` rewrites them.
The test parses both rather than comparing bytes, because `prototext` output is unstable across builds.
`-record` rewrites every text file even when only its spacing changed, so restore those that did not otherwise change.

`dotnet/Tdl.Plugin.Tests` answers each binary request with the .NET debug port and compares its bytes with the binary response, `rust/tdl-plugin-debug/tests/exchange.rs` does the same with the Rust debug port, and `js/tdl-plugin-debug/test/exchange.test.ts` with the JS one.
`catalog.tdl` sets `roundtrip`, so its request carries the model as TDL.
