module UnMango.Tdl.Testing.Spec

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! description = arbMap |> ArbMap.generate<string>
  let! name = arbMap |> ArbMap.generate<string>
  let! source = arbMap |> ArbMap.generate<string>
  let! version = arbMap |> ArbMap.generate<string>
  let! displayName = arbMap |> ArbMap.generate<string>
  // let! labels = arbMap |> ArbMap.generate<Dictionary<string, string>>
  // let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>
  // let! types = arbMap |> ArbMap.generate<Dictionary<string, Type>>

  let result =
    Spec(
      Name = name,
      Source = source,
      Version = version,
      DisplayName = displayName,
      Description = description
    )

  // TODO: Something in here is causing a stack overflow
  // result.Labels.Add(labels)
  // result.Meta.Add(meta)
  // result.Types_.Add(types)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
