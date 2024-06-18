namespace UnMango.Tdl

open System.IO
open CliWrap
open Google.Protobuf
open UnMango.CliWrap.FSharp
open UnMango.Tdl.Abstractions
open UnMango.Tdl.Tdl

module CliRunner =
  type ToolCmd = string -> Stream -> Stream -> Async<CommandResult>

  let toolCmd op tool input output = command tool {
    args [ op ]
    stdin (input :> Stream)
    stdout (PipeTo.stream output)
    async
  }

  let converter: ToolCmd = toolCmd "from"
  let generator: ToolCmd = toolCmd "gen"

  let from tool : From =
    fun input -> async {
      use stream = new MemoryStream()
      do! converter tool input stream |> Async.Ignore
      stream.Position <- 0
      let result = stream |> Spec.Parser.ParseFrom
      return Ok result
    }

  let gen tool : Gen =
    fun spec output -> async {
      use stream = new MemoryStream()
      spec.WriteTo(stream)
      stream.Position <- 0
      do! generator tool stream output |> Async.Ignore
      return None
    }

type CliRunner(tool) =
  interface IRunner with
    member _.FromAsync(input, cancellationToken) =
      Async.StartAsTask(
        async {
          match! CliRunner.from tool input with
          | Ok result -> return result
          | Error(Message m) -> return failwith m
        },
        cancellationToken = cancellationToken
      )

    member _.GenerateAsync(input, output, cancellationToken) =
      Async.StartAsTask(CliRunner.gen tool input output, cancellationToken = cancellationToken)
