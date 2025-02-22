module Ast

open Swensen.Unquote
open UnMango.Tdl
open UnMango.Tdl.Ast
open Xunit

[<Fact>]
let ``Should convert a type`` () =
  let expected = Type()
  expected.Type_ <- "Test"

  test <@ Type.proto { Name = "Test" } = Type() @>

[<Fact>]
let ``Should convert an empty spec`` () =
  test <@ Spec.proto { Types = [] } = Spec() @>

[<Fact>]
let ``Should convert a spec`` () =
  let expected = Spec()
  expected.Types_.Add([ "Test", Type() ] |> dict)

  test <@ Spec.proto { Types = [ { Name = "Test" } ] } = expected @>
