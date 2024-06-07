module UnMango.Tdl.Testing.RunnerTest

open System.IO
open UnMango.Tdl

let roundTrip (gen: Tdl.Gen, from: Tdl.From) spec = async {
  use stream = new MemoryStream()
  do! gen spec stream
  stream.Position <- 0
  let! result = from stream
  return result = spec
}
