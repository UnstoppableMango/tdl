module UnMango.Tdl.Testing.RunnerTest

open System.IO
open UnMango.Tdl

let roundTrip (gen: Tdl.Gen, from: Tdl.From) spec = async {
  use stream = new MemoryStream()

  match! gen spec stream with
  | Some(Tdl.Message e) -> return failwith e
  | _ -> ()

  stream.Position <- 0

  match! from stream with
  | Ok result -> return result.Equals(spec)
  | _ -> return false
}

let generateData (gen: Tdl.Gen, _: Tdl.From) spec = async {
  use stream = new MemoryStream()

  match! gen spec stream with
  | Some(Tdl.Message e) -> return failwith e
  | _ -> ()

  return stream.Length > 0
}

let consumeData (_: Tdl.Gen, from: Tdl.From) (spec: Spec) (stream: MemoryStream) = async {
  match! from stream with
  | Ok result -> return result.Equals(spec)
  | _ -> return false
}
