module Tests

open FsCheck.Xunit
open UnMango.Tdl
open UnMango.Tdl.Testing

module Fake =
  open Google.Protobuf
  let from: Tdl.From = fun stream -> async { return Spec.Parser.ParseFrom(stream) }
  let gen: Tdl.Gen = fun spec stream -> async { spec.WriteTo(stream) }

[<Property(Arbitrary = [| typeof<TdlArbs> |])>]
let ``Should round-trip`` spec =
  RunnerTest.roundTrip (Fake.gen, Fake.from) spec
