module UnMango.Tdl.Parser

open FParsec

type Type = { Name: string }
type Spec = { Types: Type list }

let ident: Parser<string, unit> = IdentifierOptions() |> identifier

let typ: Parser<Type, unit> =
  pstring "type" >>. spaces >>. ident |>> fun n -> { Name = n }

let spec: Parser<Spec, unit> =
  (typ |>> fun t -> { Types = [ t ] }) <|> preturn { Types = [] }

let parse = run spec
