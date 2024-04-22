namespace UnMango.Tdl

open System.IO
open UnMango.CliWrap.FSharp

type Broker = Development

module Broker =
  let private dir = Path.Join("..", "Broker")

  let dev = Development

  let start endpoint broker =
    match broker with
    | Development ->
      command "dotnet" {
        workingDirectory dir
        env [ "ASPNETCORE_URLS", endpoint ]
        args [ "run"; "--no-launch-profile" ]
      }

type Broker with
  static member Dev = Broker.dev
  member this.Start(endpoint) = Broker.start endpoint this
