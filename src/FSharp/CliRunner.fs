namespace UnMango.Tdl

open System.IO
open Google.Protobuf
open UnMango.CliWrap.FSharp
open UnMango.Tdl.Abstractions
open UnMango.Tdl.Tdl

module CliRunner =
  let from plugin : From =
    fun input -> async {
      let pluginDir = Config.Env |> Config.pluginDir
      let plugin = Path.Combine(pluginDir, plugin)
      use stream = new MemoryStream()

      let! _ = command plugin {
        args [ "from" ]
        stdin input
        stdout (PipeTo.stream stream)
        async
      }

      stream.Position <- 0
      return stream |> Spec.Parser.ParseFrom
    }

  let gen plugin : Gen =
    fun spec output -> async {
      let pluginDir = Config.Env |> Config.pluginDir
      let plugin = Path.Combine(pluginDir, plugin)
      use stream = new MemoryStream()
      spec.WriteTo(stream)
      stream.Position <- 0

      let! _ = command plugin {
        args [ "gen" ]
        stdin stream
        stdout (PipeTo.stream output)
        async
      }

      return ()
    }

type CliRunner(plugin) =
  interface IRunner with
    member this.FromAsync(input, cancellationToken) =
      Async.StartAsTask(CliRunner.from plugin input, cancellationToken = cancellationToken)

    member this.GenerateAsync(input, output, cancellationToken) =
      Async.StartAsTask(CliRunner.gen plugin input output, cancellationToken = cancellationToken)
