module UnMango.Tdl.Testing.Function

open System.Collections.Generic
open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

let generate arbMap = gen {
  let! returnType = arbMap |> ArbMap.generate<Type>
  let! parameters = arbMap |> ArbMap.generate<Dictionary<string, Type>>
  let! genericParameters = arbMap |> ArbMap.generate<Dictionary<string, GenericParameter>>
  let! meta = arbMap |> ArbMap.generate<Dictionary<string, Any>>

  let result = Function(ReturnType = returnType)

  result.Parameters.Add(parameters)
  result.GenericParameters.Add(genericParameters)
  result.Meta.Add(meta)

  return result
}

let merge: IArbMap -> IArbMap = ArbMap.mergeMapFactory (generate >> Arb.fromGen)
