namespace UnMango.Tdl.Testing

open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open UnMango.Tdl

module TdlArbs =
  let merge: IArbMap -> IArbMap =
    Any.merge
    >> Constructor.merge
    >> Field.merge
    >> Function.merge
    >> GenericParameter.merge
    >> Modifier.merge
    >> Spec.merge
    >> Type.merge

  let internal arbs = ArbMap.defaults |> merge

type TdlArbs() =
  static member Any() = TdlArbs.arbs |> ArbMap.arbitrary<Any>

  static member Constructor() =
    TdlArbs.arbs |> ArbMap.arbitrary<Constructor>

  static member Field() = TdlArbs.arbs |> ArbMap.arbitrary<Field>

  static member Function() =
    TdlArbs.arbs |> ArbMap.arbitrary<Function>

  static member GenericParameter() = TdlArbs.arbs |> ArbMap.arbitrary<Any>

  static member Modifier() =
    TdlArbs.arbs |> ArbMap.arbitrary<Modifier>

  static member Spec() = TdlArbs.arbs |> ArbMap.arbitrary<Spec>

  static member Type() = TdlArbs.arbs |> ArbMap.arbitrary<Type>
