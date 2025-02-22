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
  test <@ parse "type Test" |> success = { Types = [ { Name = "Test" } ] } @>

[<Theory>]
[<InlineData("type Test\ntype Test2")>]
[<InlineData("type Test type Test2")>]
[<InlineData("type Test\ttype Test2")>]
[<InlineData("type Test\r\ntype Test2")>]
let ``Should parse multiple empty types`` (input: string) =
  test
    <@
      parse input |> success = { Types = [ { Name = "Test" }; { Name = "Test2" } ] }
    @>
