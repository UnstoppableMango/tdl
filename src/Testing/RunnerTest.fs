module UnMango.Tdl.Testing.RunnerTest

open System.IO
open Google.Protobuf
open UnMango.Tdl

let roundTrip (gen: Tdl.Gen, from: Tdl.From) spec = async {
  use stream = new MemoryStream()
  do! gen spec stream
  stream.Position <- 0
  let! result = from stream
  return result.Equals(spec)
}

let generateData (gen: Tdl.Gen, _: Tdl.From) spec = async {
  use stream = new MemoryStream()
  do! gen spec stream
  return stream.Length > 0
}

let consumeData (_: Tdl.Gen, from: Tdl.From) (spec: Spec) (stream: MemoryStream) = async {
  let! result = from stream
  return result.Equals(spec)
}