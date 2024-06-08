module UnMango.Tdl.Testing.Constructor

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! parameters = arbMap |> ArbMap.generate<Dictionary<string, Type>>
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>

  let result = Constructor()

  result.Parameters.Add(parameters)
  result.Meta.Add(meta)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
