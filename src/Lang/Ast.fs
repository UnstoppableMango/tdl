namespace UnMango.Tdl.Ast

open UnMango.Tdl

module private Util =
  let pair key value x = (key x, value x)

type Type = { Name: string }

module Type =
  let proto x =
    let t = Type()
    t

  let internal pair = Util.pair _.Name proto

type Spec = { Types: Type list }

module Spec =
  let proto x =
    let s = Spec()
    s.Types_.Add(x.Types |> List.map Type.pair |> dict)
    s
