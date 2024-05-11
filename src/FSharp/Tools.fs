module UnMango.Tdl.Tools

open System.Text
open UnMango.CliWrap.FSharp

let id (opts: string list) = async {
  let sb = StringBuilder()

  do!
    command "id" {
      args opts
      stdout (PipeTo.string sb)
      async
    }
    |> Async.Ignore

  return sb.ToString().Trim()
}

let uid = id [ "-u" ]

let gid = id [ "-g" ]
