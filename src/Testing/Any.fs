module UnMango.Tdl.Testing.Any

open FsCheck
open FsCheck.FSharp
open Google.Protobuf.WellKnownTypes
open Google.Protobuf

let generate () = gen { return Any() }

let generatePacked arbMap = gen {
  let! url = arbMap |> ArbMap.generate<string>
  let! message = arbMap |> ArbMap.generate<IMessage>

  return Any.Pack(message, url)
}

let merge: IArbMap -> IArbMap = ArbMap.mergeFactory (generate >> Arb.fromGen)
