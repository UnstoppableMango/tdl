namespace UnMango.Tdl.Lang

open FParsec
open UnMango.Tdl.Ast

module Parser =
  let ident: Parser<string, unit> = IdentifierOptions() |> identifier

  let typ: Parser<Type, unit> =
    pstring "type" >>. spaces >>. ident |>> fun n -> { Name = n }

  let manyTypes: Parser<Type list, unit> =
    (satisfy System.Char.IsWhiteSpace) |> sepBy typ

  let spec: Parser<Spec, unit> =
    (manyTypes |>> fun t -> { Types = t }) <|> preturn { Types = [] }

  let parse = run spec

type Parser() =
  static member Parse(input) =
    match Parser.parse input with
    | Success(spec, _, _) -> spec |> Spec.proto
    | Failure(msg, _, _) -> failwith msg
