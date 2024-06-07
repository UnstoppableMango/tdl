module UnMango.Tdl.Testing.Type

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate (arbMap: IArbMap) = gen {
  let! ctor = arbMap |> ArbMap.generate<Constructor>
  let! fields = arbMap |> ArbMap.generate<Dictionary<string, Field>>
  let! genericParameters = arbMap |> ArbMap.generate<Dictionary<string, GenericParameter>>
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>
  let! methods = arbMap |> ArbMap.generate<Dictionary<string, Function>>
  let! typ = arbMap |> ArbMap.generate<string>

  let result = Type(Constructor = ctor, Type_ = typ)

  result.Fields.Add(fields)
  result.GenericParameters.Add(genericParameters)
  result.Meta.Add(meta)
  result.Methods.Add(methods)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
