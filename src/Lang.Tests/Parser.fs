module Parser

open FParsec
open Swensen.Unquote
open UnMango.Tdl.Parser
open Xunit

let success =
  function
  | Success(res, _, _) -> res
  | Failure(msg, _, _) -> failwith msg

[<Fact>]
let ``Should parse empty spec`` () =
  test <@ parse "" |> success = { Types = [] } @>

[<Fact>]
let ``Should parse an empty type`` () =
  test <@ parse "type Test" |> success = { Types = [{ Name = "Test" }] } @>
