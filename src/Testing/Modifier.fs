module UnMango.Tdl.Testing.Modifier

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>

  let result = Modifier()

  result.Meta.Add(meta)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
