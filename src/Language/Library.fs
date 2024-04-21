namespace UnMango.Tdl.Language

type Primitive =
  | String
  | Integer
  | Boolean

type Object = Map<string, Primitive>
