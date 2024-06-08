module UnMango.Tdl.Testing.Field

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! typ = arbMap |> ArbMap.generate<string>
  let! ro = arbMap |> ArbMap.generate<bool>
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>

  let result = Field(Type = typ, Readonly = ro)

  result.Meta.Add(meta)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
