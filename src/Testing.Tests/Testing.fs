module Testing

open System.IO
open FsCheck.Xunit
open Google.Protobuf
open UnMango.Tdl
open UnMango.Tdl.Testing

module Fake =
  let from: Tdl.From = fun stream -> async { return Spec.Parser.ParseFrom(stream) }
  let gen: Tdl.Gen = fun spec stream -> async { spec.WriteTo(stream) }

[<Property(Arbitrary = [| typeof<TdlArbs> |])>]
let ``Should round-trip`` spec =
  RunnerTest.roundTrip (Fake.gen, Fake.from) spec

[<Property(Arbitrary = [| typeof<TdlArbs> |])>]
let ``Should generate data`` spec =
  RunnerTest.generateData (Fake.gen, Fake.from) spec

[<Property(Arbitrary = [| typeof<TdlArbs> |])>]
let ``Should consume data`` (spec: Spec) =
  let stream = new MemoryStream()
  spec.WriteTo(stream)
  stream.Position <- 0
  RunnerTest.consumeData (Fake.gen, Fake.from) spec stream
