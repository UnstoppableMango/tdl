module UnMango.Tdl.Testing.Spec

open FsCheck.FSharp
open UnMango.Tdl

let gen arbs = gen {
  let! description = arbs |> ArbMap.generate<string>
  let! name = arbs |> ArbMap.generate<string>
  let! source = arbs |> ArbMap.generate<string>
  let! version = arbs |> ArbMap.generate<string>
  let! displayName = arbs |> ArbMap.generate<string>

  return
    Spec(
      Description = description,
      Name = name,
      Source = source,
      Version = version,
      DisplayName = displayName
    )
}
