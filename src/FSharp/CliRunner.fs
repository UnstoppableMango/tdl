namespace UnMango.Tdl

open System.IO
open UnMango.CliWrap.FSharp
open UnMango.Tdl.Abstractions
open UnMango.Tdl.Tdl

module CliRunner =
  let from: From =
    fun input -> async {
      let! ct = Async.CancellationToken

      let pluginDir = Config.Env |> Config.pluginDir
      let plugin = Path.Combine(pluginDir, "TODO")

      let! res = command plugin {
        args [ "from" ]
        async
      }

      return Spec()
    }

  let gen: Gen = failwith "TODO"

type CliRunner() =
  interface IRunner with
    member this.FromAsync(input, cancellationToken) =
      Async.StartAsTask(CliRunner.from input, cancellationToken = cancellationToken)

    member this.GenerateAsync(input, output, cancellationToken) =
      Async.StartAsTask(CliRunner.gen input output, cancellationToken = cancellationToken)
