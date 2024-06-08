module UnMango.Tdl.Testing.GenericParameter

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! modifiers = arbMap |> ArbMap.generate<Modifier list>
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>

  let result = GenericParameter()

  result.Meta.Add(meta)
  result.Modifiers.Add(modifiers)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
