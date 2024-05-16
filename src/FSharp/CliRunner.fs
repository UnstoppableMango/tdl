namespace UnMango.Tdl

open System.IO
open Google.Protobuf
open UnMango.CliWrap.FSharp
open UnMango.Tdl.Abstractions
open UnMango.Tdl.Tdl

module CliRunner =
  let from: From =
    fun input -> async {
      let pluginDir = Config.Env |> Config.pluginDir
      let plugin = Path.Combine(pluginDir, source.Plugin)
      use stream = new MemoryStream()

      let! _ = command plugin {
        args [ "from" ]
        stdin source.Input
        stdout (PipeTo.stream stream)
        async
      }

      stream.Position <- 0
      return stream |> Spec.Parser.ParseFrom |> Ok
    }

  let gen: Gen =
    fun spec target -> async {
      let pluginDir = Config.Env |> Config.pluginDir
      let plugin = Path.Combine(pluginDir, target.Plugin)
      use stream = new MemoryStream()
      spec.WriteTo(stream)
      stream.Position <- 0

      let! _ = command plugin {
        args [ "gen" ]
        stdin stream
        stdout (PipeTo.stream target.Output)
        async
      }

      return Ok()
    }

type CliRunner() =
  interface IRunner with
    member this.FromAsync(input, cancellationToken) =
      Async.StartAsTask(CliRunner.from input, cancellationToken = cancellationToken)

    member this.GenerateAsync(input, output, cancellationToken) =
      Async.StartAsTask(CliRunner.gen input output, cancellationToken = cancellationToken)
