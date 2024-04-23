namespace UnMango.Tdl

open System.IO
open UnMango.CliWrap.FSharp

type DevelopmentMeta = { Configuration: string; Tfm: string }

type BrokerLocation = Development of DevelopmentMeta

type Broker =
  { Endpoint: string
    Location: BrokerLocation
    Bin: string }

module Broker =
  let private dir = Path.Join("..", "Broker")

  let dev = Development

  let binDir broker =
    match broker.Location with
    | Development meta -> Path.Join(dir, "bin", meta.Configuration, meta.Tfm)

  let buildCmd meta =
    command "dotnet" {
      workingDirectory dir
      args [ "build" ]
    }

  let startCmd binDir broker =
    match broker.Location with
    | Development meta ->
      command "dotnet" {
        workingDirectory binDir
        env [ "ASPNETCORE_URLS", broker.Endpoint ]
        args [ broker.Bin ]
      }

  let start broker =
    async {
      match broker.Location with
      | Development _ ->
        let! _ = buildCmd broker |> Cli.exec
        let binDir = binDir broker
        let! _ = startCmd binDir broker |> Cli.exec
        return broker
    }

type Broker with
  static member Dev = Broker.dev
  member this.Start(endpoint) = Broker.start endpoint this
