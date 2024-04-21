namespace UnMango.Tdl.Language

open FParsec

type Primitive =
  | String
  | Number
  | Boolean
  | Void

type Object = Map<string, Value>

and Value =
  | Primitive of Primitive
  | Object of Object
  | Generic of Object * Value list
  | Union of Value list
  | List of Value list

type Uml =
  | Version of string
  | Objects of Map<string, Object>

type Schema =
  { Objects: Map<string, Object>
    Version: string }

type SchemaParser<'a> = Parser<Schema, 'a>

module Yaml =
  let bare = { Objects = Map.empty; Version = "0.1" }

  let str: Parser<Primitive, string> = stringReturn "string" String
  let num: Parser<Primitive, string> = stringReturn "number" Number
  let bool: Parser<Primitive, string> = stringReturn "boolean" Boolean
  let vd: Parser<Primitive, string> = stringReturn "void" Void

  let typ = choice [ str ]

  let run v =
    runParserOnString str "" "" v
    |> function
      | Success(result, _, _) -> Result.Ok result
      | Failure(msg, _, _) -> Result.Error msg
