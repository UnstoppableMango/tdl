namespace UnMango.Tdl.Language

open FParsec

type Primitive =
  | String
  | Integer
  | Boolean

type Object = Map<string, Primitive>

type Schema =
  { Objects: Map<string, Object>
    Version: string }

type SchemaParser<'a> = Parser<Schema, 'a>

module Uml =
  let run x y =
    match CharParsers.run x y with
    | Success(foo, unit, position) -> failwith "todo"
    | Failure(s, parserError, unit) -> failwith "todo"
